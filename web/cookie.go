package web

import (
	"net/http"
	"strconv"
)

// CookieBuilder fluently constructs and writes a cookie.
type CookieBuilder struct {
	c http.Cookie
}

// Cookie starts a builder for name=value.
func Cookie(name, value string) *CookieBuilder {
	return &CookieBuilder{c: http.Cookie{Name: name, Value: value, Path: "/"}}
}

// Path restricts the cookie to a URL path prefix.
func (b *CookieBuilder) Path(p string) *CookieBuilder { b.c.Path = p; return b }

// Domain scopes the cookie to a domain.
func (b *CookieBuilder) Domain(d string) *CookieBuilder { b.c.Domain = d; return b }

// MaxAge sets lifetime in seconds (negative deletes).
func (b *CookieBuilder) MaxAge(seconds int) *CookieBuilder { b.c.MaxAge = seconds; return b }

// Secure sends the cookie over HTTPS only.
func (b *CookieBuilder) Secure() *CookieBuilder { b.c.Secure = true; return b }

// HTTPOnly hides the cookie from JavaScript.
func (b *CookieBuilder) HTTPOnly() *CookieBuilder { b.c.HttpOnly = true; return b }

// SameSite sets the cross-site policy (Lax default in browsers).
func (b *CookieBuilder) SameSite(s http.SameSite) *CookieBuilder { b.c.SameSite = s; return b }

// Write attaches the cookie to the response.
func (b *CookieBuilder) Write(w http.ResponseWriter) {
	http.SetCookie(w, &b.c)
}

// DeleteCookie clears a cookie on the client.
func DeleteCookie(w http.ResponseWriter, name, path string) {
	if path == "" {
		path = "/"
	}
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     path,
		MaxAge:   -1,
		HttpOnly: true,
	})
}

// CookieVal returns the named request cookie's value, "" when absent.
func (r *Request) CookieVal(name string) string {
	return cookieVal(r.Request, name)
}

func cookieVal(r *http.Request, name string) string {
	c, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}

func strconvItoa(n int) string { return strconv.Itoa(n) }
