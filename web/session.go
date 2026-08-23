package web

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/krewire/framework/storage"
)

// Session is a server-side session: an opaque ID plus JSON-serializable data.
type Session struct {
	ID      string
	data    map[string]any
	exp     time.Time
	dirty   bool
	rotated bool // old id pending deletion in store
	oldID   string
}

// Get returns the value for key and whether it existed.
func (s *Session) Get(key string) (any, bool) {
	v, ok := s.data[key]
	return v, ok
}

// Set stores v under key, marking the session dirty.
func (s *Session) Set(key string, v any) {
	if s.data == nil {
		s.data = map[string]any{}
	}
	s.data[key] = v
	s.dirty = true
}

// Delete removes key.
func (s *Session) Delete(key string) {
	delete(s.data, key)
	s.dirty = true
}

// Rotate regenerates the session ID, keeping data — the fixation defense.
// The previous record is deleted on save.
func (s *Session) Rotate() {
	if s.rotated {
		return
	}
	s.oldID = s.ID
	s.ID = randomToken()
	s.rotated = true
	s.dirty = true
}

type sessionRecord struct {
	Data map[string]any `json:"data"`
	Exp  time.Time      `json:"exp"`
}

// SessionStore persists sessions by ID. Implementations must treat expired
// or unknown IDs as absent.
type SessionStore interface {
	Load(id string) (*sessionRecord, bool, error)
	Save(id string, rec *sessionRecord) error
	Delete(id string) error
}

// MemorySessionStore keeps sessions in process memory.
type MemorySessionStore struct {
	mu    sync.RWMutex
	items map[string]*sessionRecord
}

// NewMemorySessionStore returns an empty in-process store.
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{items: map[string]*sessionRecord{}}
}

func (m *MemorySessionStore) Load(id string) (*sessionRecord, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rec, ok := m.items[id]
	if !ok || time.Now().After(rec.Exp) {
		return nil, false, nil
	}
	cp := *rec
	return &cp, true, nil
}

func (m *MemorySessionStore) Save(id string, rec *sessionRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *rec
	m.items[id] = &cp
	return nil
}

func (m *MemorySessionStore) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, id)
	return nil
}

// KVSessionStore adapts any storage.KV backend, namespacing keys under
// sessions/.
type KVSessionStore struct{ kv storage.KV }

// NewKVSessionStore backs sessions with the given KV store.
func NewKVSessionStore(kv storage.KV) *KVSessionStore { return &KVSessionStore{kv: kv} }

func keyFor(id string) string { return "sessions/" + strings.ReplaceAll(id, "/", "_") }

func (k *KVSessionStore) Load(id string) (*sessionRecord, bool, error) {
	raw, ok, err := k.kv.Get(context.Background(), keyFor(id))
	if err != nil || !ok {
		return nil, false, err
	}
	var rec sessionRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		return nil, false, err
	}
	if time.Now().After(rec.Exp) {
		_ = k.kv.Delete(context.Background(), keyFor(id))
		return nil, false, nil
	}
	return &rec, true, nil
}

func (k *KVSessionStore) Save(id string, rec *sessionRecord) error {
	raw, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return k.kv.Put(context.Background(), keyFor(id), raw)
}

func (k *KVSessionStore) Delete(id string) error {
	return k.kv.Delete(context.Background(), keyFor(id))
}

// SessionOptions tunes the sessions middleware.
type SessionOptions struct {
	// CookieName defaults to "kiw_session".
	CookieName string
	// IdleTTL is the sliding lifetime; default 24h.
	IdleTTL time.Duration
	// Secure marks the cookie HTTPS-only.
	Secure bool
	// SameSite overrides Lax.
	SameSite http.SameSite
}

type sessionCtxKey struct{}

// Sessions resolves (or creates) the caller's session before the handler and
// persists it afterwards when dirty, refreshing the sliding expiry.
func Sessions(store SessionStore, opts ...func(*SessionOptions)) Middleware {
	o := &SessionOptions{}
	for _, f := range opts {
		f(o)
	}
	if o.CookieName == "" {
		o.CookieName = "kiw_session"
	}
	if o.IdleTTL <= 0 {
		o.IdleTTL = 24 * time.Hour
	}
	if o.SameSite == http.SameSiteDefaultMode {
		o.SameSite = http.SameSiteLaxMode
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess := resolveSession(store, cookieVal(r, o.CookieName), o.IdleTTL)
			sess.exp = time.Now().Add(o.IdleTTL)

			ck := &http.Cookie{
				Name:     o.CookieName,
				Value:    sess.ID,
				Path:     "/",
				HttpOnly: true,
				Secure:   o.Secure,
				SameSite: o.SameSite,
			}
			sw := &statusRecorder{ResponseWriter: w}
			http.SetCookie(sw, ck)

			r = r.WithContext(context.WithValue(r.Context(), sessionCtxKey{}, sess))
			next.ServeHTTP(sw, r)

			if sess.dirty {
				if err := store.Save(sess.ID, &sessionRecord{Data: sess.data, Exp: sess.exp}); err == nil {
					sess.dirty = false
				}
			}
			if sess.rotated && sess.oldID != "" {
				_ = store.Delete(sess.oldID)
				sess.rotated = false
				if sw.status == 0 { // headers still open — swap in the new ID
					dropCookies(sw.Header(), o.CookieName)
					ck.Value = sess.ID
					http.SetCookie(sw, ck)
				}
			}
		})
	}
}

// dropCookies removes prior Set-Cookie lines for one cookie name.
func dropCookies(h http.Header, name string) {
	old := h.Values("Set-Cookie")
	h.Del("Set-Cookie")
	for _, v := range old {
		if strings.HasPrefix(v, name+"=") {
			continue
		}
		h.Add("Set-Cookie", v)
	}
}

func resolveSession(store SessionStore, cookieID string, ttl time.Duration) *Session {
	if cookieID != "" {
		if rec, ok, _ := store.Load(cookieID); ok {
			return &Session{ID: cookieID, data: rec.Data, exp: rec.Exp}
		}
	}
	// New session: dirty so the cookie is written on this response.
	return &Session{ID: randomToken(), data: map[string]any{}, dirty: true}
}

// SessionFrom returns the request's session, nil before the middleware ran.
func SessionFrom(ctx context.Context) *Session {
	v, _ := ctx.Value(sessionCtxKey{}).(*Session)
	return v
}

// Session returns the request's session for expressive handlers.
func (r *Request) Session() *Session { return SessionFrom(r.Context()) }
