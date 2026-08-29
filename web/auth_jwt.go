package web

import "github.com/krewire/libs/auth"

// Claims is a JWT claim set.
//
// Deprecated: use auth.Claims.
type Claims = auth.Claims

// ErrInvalidToken classifies any rejected JWT.
//
// Deprecated: use auth.ErrInvalidToken.
var ErrInvalidToken = auth.ErrInvalidToken

// SignJWT produces a compact HS256 JWS.
//
// Deprecated: use auth.SignJWT.
var SignJWT = auth.SignJWT

// DefaultClaims seeds standard claims.
//
// Deprecated: use auth.DefaultClaims.
var DefaultClaims = auth.DefaultClaims

// ParseJWT verifies signature and expiry.
//
// Deprecated: use auth.ParseJWT.
var ParseJWT = auth.ParseJWT

// JWTOptions tunes the JWTAuth middleware.
//
// Deprecated: use auth.JWTOptions.
type JWTOptions = auth.JWTOptions

// ClaimCheck asserts one claim equality.
//
// Deprecated: use auth.ClaimCheck.
type ClaimCheck = auth.ClaimCheck

// JWTAuth verifies bearer credentials and stores the identity.
//
// Deprecated: use auth.JWTAuth.
func JWTAuth(secret []byte, opts ...func(*JWTOptions)) Middleware {
	return Middleware(auth.JWTAuth(secret, opts...))
}
