package web

import "github.com/krewire/libs/auth"

// BasicVerifier validates an identifier/password pair.
//
// Deprecated: use auth.BasicVerifier.
type BasicVerifier = auth.BasicVerifier

// BasicAuth implements RFC 7617 over the verifier.
//
// Deprecated: use auth.BasicAuth.
func BasicAuth(realm string, verify BasicVerifier) Middleware {
	return Middleware(auth.BasicAuth(realm, auth.BasicVerifier(verify)))
}
