// Tests for KWF-WEB-R9T4C
package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/krewire/framework/storage"
	ftest "github.com/krewire/framework/test"
)

func headerValues(h http.Header, key string) []string { return h.Values(key) }

// Spec: KWF-WEB-R9T4C FRK-SEC-001 Scope: Package
func TestFRK_SEC_001_SecurityHeaders_DefaultsAndOverrides(t *testing.T) {
	r := NewRouter()
	r.Use(SecurityHeaders(func(o *SecurityOptions) {
		o.HSTS = 63072000
		o.Frame = "SAMEORIGIN"
	}))
	r.Get("/", func(w http.ResponseWriter, req *http.Request, p Params) { w.WriteHeader(200) })

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, ftest.NewRequest(t, http.MethodGet, "/", nil))

	for k, want := range map[string]string{
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "SAMEORIGIN",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
		"Content-Security-Policy":   "default-src 'self'",
		"Strict-Transport-Security": "max-age=63072000; includeSubDomains",
	} {
		if got := rec.Header().Get(k); got != want {
			t.Errorf("%s = %q want %q", k, got, want)
		}
	}
}

// Spec: KWF-WEB-R9T4C FRK-SEC-002 Scope: Package
func TestFRK_SEC_002_StripTags(t *testing.T) {
	ftest.Equal(t, "hello world", StripTags("<b>hello</b> <i>world</i>"))
	ftest.Equal(t, "no tags", StripTags("  no tags  "))
}

// csrfStack builds router with security+session+csrf and a probe handler that
// records session state.
type probe struct {
	gotSession bool
	token      string
	sessID     string
}

type seenIn struct{}

func csrfStack(t *testing.T, store SessionStore) (*Router, *probe, *CSRFOptions) {
	t.Helper()
	o := &CSRFOptions{}
	r := NewRouter()
	r.Use(SecurityHeaders())
	r.Use(Sessions(store))
	r.Use(CSRF(func(c *CSRFOptions) { *o = *c }))
	return r, &probe{}, o
}

type submitBody struct {
	Name string `json:"name"`
}

// Spec: KWF-WEB-R9T4C FRK-SEC-010+011 Scope: Package
func TestFRK_SEC_010_CSRF_FlowWithSession(t *testing.T) {
	store := NewMemorySessionStore()
	r, p, _ := csrfStack(t, store)
	r.Post("/submit", H(func(req *Request, in *submitBody) (any, error) {
		p.gotSession = req.Session() != nil
		p.token = req.CSRFToken()
		req.Session().Set("seen", true)
		return map[string]string{"ok": "1"}, nil
	}))

	// Step 1: GET issues both cookies (session + csrf token)
	get1 := httptest.NewRecorder()
	r.ServeHTTP(get1, ftest.NewRequest(t, http.MethodGet, "/submit", nil))
	csrfCookie := findCookie(get1, "XSRF-TOKEN")
	if csrfCookie == nil {
		t.Fatal("no csrf cookie issued")
	}
	sessCookie := findCookie(get1, "kiw_session")
	if sessCookie == nil {
		t.Fatal("no session cookie issued")
	}

	// Step 2: POST without token -> 403
	post := ftest.NewRequest(t, http.MethodPost, "/submit", strings.NewReader(`{}`))
	post.Header.Set("Content-Type", "application/json")
	post.AddCookie(csrfCookie)
	post.AddCookie(sessCookie)
	post2 := httptest.NewRecorder()
	r.ServeHTTP(post2, post)
	ftest.EqualStatus(t, post2, http.StatusForbidden)

	// Step 3: POST with token in header -> 200, session bound token visible
	post3 := ftest.NewRequest(t, http.MethodPost, "/submit", strings.NewReader(`{"name":"a"}`))
	post3.Header.Set("Content-Type", "application/json")
	post3.AddCookie(csrfCookie)
	post3.AddCookie(sessCookie)
	post3.Header.Set("X-CSRF-Token", csrfCookie.Value)
	rec3 := httptest.NewRecorder()
	r.ServeHTTP(rec3, post3)
	ftest.EqualStatus(t, rec3, http.StatusOK)
	if !p.gotSession || p.token == "" {
		t.Errorf("probe session=%v token=%q", p.gotSession, p.token)
	}

	// Step 4: API-style request with Authorization skips CSRF
	post4 := ftest.NewRequest(t, http.MethodPost, "/submit", strings.NewReader(`{"name":"x"}`))
	post4.Header.Set("Authorization", "Bearer z")
	post4.Header.Set("Content-Type", "application/json")
	rec4 := httptest.NewRecorder()
	r.ServeHTTP(rec4, post4)
	ftest.EqualStatus(t, rec4, http.StatusOK)
}

// Spec: KWF-WEB-R9T4C FRK-SEC-032+033 Scope: Package
func TestFRK_SEC_032_Session_Rotate_KV_vs_Memory(t *testing.T) {
	kvStore := NewKVSessionStore(storage.NewMemory())
	memStore := NewMemorySessionStore()
	for name, store := range map[string]SessionStore{"kv": kvStore, "memory": memStore} {
		t.Run(name, func(t *testing.T) {
			r := NewRouter()
			r.Use(Sessions(store))
			var firstID string
			r.Get("/start", func(w http.ResponseWriter, req *http.Request, p Params) {
				s := SessionFrom(req.Context())
				s.Set("user", "kiw")
				firstID = s.ID
			})
			r.Get("/rotate", func(w http.ResponseWriter, req *http.Request, p Params) {
				SessionFrom(req.Context()).Rotate()
			})
			r.Get("/who", func(w http.ResponseWriter, req *http.Request, p Params) {
				s := SessionFrom(req.Context())
				if v, ok := s.Get("user"); ok {
					w.Write([]byte(v.(string)))
				}
			})

			c1 := httptest.NewRecorder()
			r.ServeHTTP(c1, ftest.NewRequest(t, http.MethodGet, "/start", nil))
			sc := findCookie(c1, "kiw_session")
			if sc == nil || sc.Value != firstID {
				t.Fatalf("session cookie missing/mismatched: %v", sc)
			}

			c2 := httptest.NewRecorder()
			req2, _ := http.NewRequest(http.MethodGet, "/rotate", nil)
			req2.Header.Set("Cookie", "kiw_session="+sc.Value)
			r.ServeHTTP(c2, req2)
			newSc := findCookie(c2, "kiw_session")
			if newSc == nil || newSc.Value == sc.Value {
				t.Fatalf("rotate did not issue new id: %v", newSc)
			}

			c3 := httptest.NewRecorder()
			req3, _ := http.NewRequest(http.MethodGet, "/who", nil)
			req3.Header.Set("Cookie", "kiw_session="+newSc.Value)
			r.ServeHTTP(c3, req3)
			if c3.Body.String() != "kiw" {
				t.Errorf("data lost across rotate: %q", c3.Body.String())
			}

			// old id must be dead
			c4 := httptest.NewRecorder()
			req4, _ := http.NewRequest(http.MethodGet, "/who", nil)
			req4.Header.Set("Cookie", "kiw_session="+sc.Value)
			r.ServeHTTP(c4, req4)
			if c4.Body.String() != "" {
				t.Errorf("old id still alive after rotate: %q", c4.Body.String())
			}
		})
	}

}

// Spec: KWF-WEB-R9T4C FRK-SEC-031 Scope: Package
func TestFRK_SEC_031_Session_ExpiryLazy(t *testing.T) {
	store := NewMemorySessionStore()
	id := "expiring-id"
	_ = store.Save(id, &sessionRecord{
		Data: map[string]any{"k": "v"},
		Exp:  time.Now().Add(-time.Minute),
	})
	if _, ok, _ := store.Load(id); ok {
		t.Fatal("expired record must be absent")
	}
}

// Spec: KWF-WEB-R9T4C FRK-SEC-020 Scope: Package
func TestFRK_SEC_020_CachePolicies(t *testing.T) {
	cases := []struct {
		name string
		mw   Middleware
		want string
	}{
		{"nostore", NoStore(), "no-store, private"},
		{"maxage-private", MaxAge(60, false), "private, max-age=60"},
		{"maxage-public", MaxAge(300, true), "public, max-age=300"},
		{"immutable", Immutable(31536000), "public, max-age=31536000, immutable"},
		{"raw", CacheControl("no-cache"), "no-cache"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRouter()
			r.Use(tc.mw)
			r.Get("/", func(w http.ResponseWriter, req *http.Request, p Params) {})
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, ftest.NewRequest(t, http.MethodGet, "/", nil))
			if got := rec.Header().Get("Cache-Control"); got != tc.want {
				t.Errorf("Cache-Control = %q want %q", got, tc.want)
			}
		})
	}
}

// Spec: KWF-WEB-R9T4C FRK-SEC-040 Scope: Package
func TestFRK_SEC_040_CookieBuilderAndVal(t *testing.T) {
	r := NewRouter()
	r.Get("/set", func(w http.ResponseWriter, req *http.Request, p Params) {
		Cookie("theme", "dark").Path("/").MaxAge(86400).Secure().HTTPOnly().SameSite(http.SameSiteStrictMode).Write(w)
	})
	r.Get("/read", func(w http.ResponseWriter, req *http.Request, p Params) {
		w.Write([]byte(Wrap(req, p).CookieVal("theme")))
	})
	r.Get("/clear", func(w http.ResponseWriter, req *http.Request, p Params) {
		DeleteCookie(w, "theme", "/")
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, ftest.NewRequest(t, http.MethodGet, "/set", nil))
	sc := findCookie(rec, "theme")
	if sc == nil || sc.Value != "dark" || !sc.Secure || !sc.HttpOnly || sc.MaxAge != 86400 || sc.SameSite != http.SameSiteStrictMode {
		t.Fatalf("cookie attrs wrong: %+v", sc)
	}

	rec2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/read", nil)
	req2.AddCookie(sc)
	r.ServeHTTP(rec2, req2)
	ftest.Equal(t, "dark", rec2.Body.String())

	rec3 := httptest.NewRecorder()
	r.ServeHTTP(rec3, ftest.NewRequest(t, http.MethodGet, "/clear", nil))
	dc := findCookie(rec3, "theme")
	if dc == nil || dc.MaxAge != -1 {
		t.Errorf("delete cookie wrong: %+v", dc)
	}
}

func findCookie(rec *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func cookieLine(c *http.Cookie) string { return c.Name + "=" + c.Value }
