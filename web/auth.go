package web

import (
	"context"
	"net/http"
	"strings"

	"github.com/krewire/libs/auth"
)

// Identity is the authenticated caller.
//
// Deprecated: use auth.Identity.
type Identity = auth.Identity

// IdentityFrom returns the request identity, nil when anonymous.
func IdentityFrom(ctx context.Context) *auth.Identity { return auth.IdentityFrom(ctx) }

// Identity returns the request identity for expressive handlers.
func (r *Request) Identity() *auth.Identity { return auth.IdentityFrom(r.Context()) }

func withIdentity(ctx context.Context, id *auth.Identity) context.Context {
	return auth.WithIdentity(ctx, id)
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

// Ensure auth helpers are available for web's policy
var _ = auth.IdentityFrom
var _ = http.StatusUnauthorized
var _ = strings.SplitN
