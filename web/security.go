package web

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"regexp"

	"github.com/krewire/libs/sec"
)

// SecurityOptions tunes the security-headers middleware.
//
// Deprecated: use sec.SecurityOptions.
type SecurityOptions = sec.SecurityOptions

var tagStripper = regexp.MustCompile(`<[^>]*>`)

// StripTags removes HTML tags from s — delegates to sec as heart of defense.
func StripTags(s string) string {
	return sec.StripTags(s)
}

// SecurityHeaders returns middleware applying browser hardening headers.
// Delegates to sec as heart of defense; web re-exports for backward compat.
func SecurityHeaders(opts ...func(*SecurityOptions)) Middleware {
	return Middleware(sec.SecurityHeaders(opts...))
}

func setOnce(h http.Header, key, val string) {
	if h.Get(key) == "" {
		h.Set(key, val)
	}
}

// CSRFOptions tunes CSRF protection.
type CSRFOptions struct {
	// CookieName stores the token; default "XSRF-TOKEN".
	CookieName string
	// HeaderName supplies the token on unsafe requests; default
	// "X-CSRF-Token".
	HeaderName string
	// FieldName is the alternate form-field name; default "csrf_token".
	FieldName string
	// Secure marks the cookie Secure.
	Secure bool
	// HTTPOnly keeps JS from reading the cookie when true (default false so
	// SPAs can echo it back).
	HTTPOnly bool
}

type csrfCtxKey struct{}

// CSRF returns double-submit token middleware. Safe methods ensure a token
// cookie exists; unsafe methods verify header or form token against the
// cookie (or the session-bound token when sessions run first). Requests with
// an Authorization header are treated as API calls and skipped.
func CSRF(opts ...func(*CSRFOptions)) Middleware {
	o := &CSRFOptions{}
	for _, f := range opts {
		f(o)
	}
	if o.CookieName == "" {
		o.CookieName = "XSRF-TOKEN"
	}
	if o.HeaderName == "" {
		o.HeaderName = "X-CSRF-Token"
	}
	if o.FieldName == "" {
		o.FieldName = "csrf_token"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := cookieVal(r, o.CookieName)
			safe := r.Method == http.MethodGet || r.Method == http.MethodHead ||
				r.Method == http.MethodOptions || r.Method == http.MethodTrace

			if token == "" && safe && r.Header.Get("Authorization") == "" {
				token = randomToken()
				http.SetCookie(w, &http.Cookie{
					Name:     o.CookieName,
					Value:    token,
					Path:     "/",
					Secure:   o.Secure,
					HttpOnly: o.HTTPOnly,
					SameSite: http.SameSiteLaxMode,
				})
			}

			if !safe && r.Header.Get("Authorization") == "" {
				submitted := r.Header.Get(o.HeaderName)
				if submitted == "" {
					_ = r.ParseForm()
					submitted = r.PostForm.Get(o.FieldName)
				}
				expected := token
				if sess := SessionFrom(r.Context()); sess != nil {
					if v, ok := sess.Get("csrf"); ok {
						if s, ok2 := v.(string); ok2 {
							expected = s
						}
					} else if token != "" {
						sess.Set("csrf", token)
						sess.dirty = true
						expected = token
					}
				}
				if submitted == "" || expected == "" || !constantTimeEqual(submitted, expected) {
					Error(w, Forbidden("csrf token mismatch"))
					return
				}
			}

			ctx := withCSRF(r.Context(), token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CSRFFrom returns the current request's CSRF token ("" before the
// middleware ran).
func CSRFFrom(ctx context.Context) string {
	v, _ := ctx.Value(csrfCtxKey{}).(string)
	return v
}

// CSRFToken exposes the current token to expressive handlers and templates.
func (r *Request) CSRFToken() string {
	return CSRFFrom(r.Context())
}

func withCSRF(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, csrfCtxKey{}, token)
}

// constantTimeEqual compares two tokens without leaking length/timing.
func constantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("web: crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}
