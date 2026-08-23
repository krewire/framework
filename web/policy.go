package web

import "net/http"

// Policy is a before-gate: return an error (typically *HTTPError) to reject
// the request before it reaches the handler.
type Policy func(r *Request) error

// Require returns middleware running policies in order; the first failure is
// mapped through Error() so HTTPError policies control their status codes.
func Require(policies ...Policy) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			req := Wrap(r, nil)
			for _, p := range policies {
				if err := p(req); err != nil {
					Error(w, err)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// AfterRequest observes the finished response: fn receives the request and the
// final status (0 when the handler never wrote a status explicitly).
func AfterRequest(fn func(r *http.Request, status int)) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sw := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(sw, r)
			fn(r, sw.status)
		})
	}
}

// Authenticated requires any identity (401 otherwise).
func Authenticated() Policy {
	return func(r *Request) error {
		if IdentityFrom(r.Context()) == nil {
			return Unauthorized("authentication required")
		}
		return nil
	}
}

// WithRoles requires the identity to carry at least one of roles (403
// otherwise, 401 when anonymous).
func WithRoles(roles ...string) Policy {
	return func(r *Request) error {
		id := IdentityFrom(r.Context())
		if id == nil {
			return Unauthorized("authentication required")
		}
		for _, want := range roles {
			if id.HasRole(want) {
				return nil
			}
		}
		return Forbidden("insufficient role")
	}
}

// PolicySet names policies once and references them declaratively:
//
//	var gates = web.PolicySet{"admin": web.WithRoles("admin")}
//	r.Group("/admin", jwtMW, gates.Require("admin"))
type PolicySet map[string]Policy

// Require resolves the named policies in order. Unknown names panic — a
// developer error fixed at first run, never a runtime path.
func (ps PolicySet) Require(names ...string) Middleware {
	policies := make([]Policy, len(names))
	for i, n := range names {
		p, ok := ps[n]
		if !ok {
			panic("web: unknown policy " + n)
		}
		policies[i] = p
	}
	return Require(policies...)
}
