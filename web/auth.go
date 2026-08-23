package web

import (
	"context"
	"net/http"
	"strings"
)

// Identity is the authenticated caller. Method names the credential scheme
// ("basic" or "jwt"); Claims carries raw verified claims for JWT.
type Identity struct {
	// Subject is the stable caller identifier (basic identifier / JWT "sub").
	Subject string
	// Method is the credential scheme that produced the identity.
	Method string
	// Roles are plain role strings merged from credentials.
	Roles []string
	// Claims holds raw JWT claims; nil for basic auth.
	Claims map[string]any
}

// HasRole reports whether the identity carries the role.
func (id *Identity) HasRole(role string) bool {
	for _, r := range id.Roles {
		if r == role {
			return true
		}
	}
	return false
}

type identityCtxKey struct{}

// IdentityFrom returns the request identity, nil when anonymous.
func IdentityFrom(ctx context.Context) *Identity {
	id, _ := ctx.Value(identityCtxKey{}).(*Identity)
	return id
}

// Identity returns the request identity for expressive handlers.
func (r *Request) Identity() *Identity { return IdentityFrom(r.Context()) }

func withIdentity(ctx context.Context, id *Identity) context.Context {
	return context.WithValue(ctx, identityCtxKey{}, id)
}

// Unauthorized returns a 401 HTTPError.
func Unauthorized(message string) *HTTPError {
	return &HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: message}
}

// authParam splits an Authorization header into scheme and parameter.
func authParam(h string) (scheme, param string, ok bool) {
	if h == "" {
		return "", "", false
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}
