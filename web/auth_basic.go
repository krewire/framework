package web

import (
	"crypto/subtle"
	"encoding/base64"
	"net/http"
)

// BasicVerifier validates an identifier/password pair. Return the identity on
// success; return (nil, nil) or an error to reject.
type BasicVerifier func(identifier, password string) (*Identity, error)

// BasicAuth implements RFC 7617 over the verifier. Failures answer
// `401` with a Basic challenge for the given realm; the verifier is not
// invoked for malformed headers.
func BasicAuth(realm string, verify BasicVerifier) Middleware {
	challenge := "Basic realm=\"" + realm + "\""
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scheme, param, ok := authParam(r.Header.Get("Authorization"))
			if !ok || !strEqFold(scheme, "Basic") {
				challenge401(w, challenge)
				return
			}
			id, pass, err := decodeBasicPair(param)
			if err != nil {
				challenge401(w, challenge)
				return
			}
			identity, verr := verify(id, pass)
			if verr != nil || identity == nil {
				if verr != nil {
					Error(w, verr)
					return
				}
				challenge401(w, challenge)
				return
			}
			identity.Method = "basic"
			next.ServeHTTP(w, r.WithContext(withIdentity(r.Context(), identity)))
		})
	}
}

func challenge401(w http.ResponseWriter, challenge string) {
	w.Header().Set("WWW-Authenticate", challenge)
	Error(w, Unauthorized("authentication required"))
}

func decodeBasicPair(param string) (identifier, password string, err error) {
	raw, err := base64.StdEncoding.DecodeString(param)
	if err != nil {
		// Some clients send unpadded base64; retry tolerant decode.
		raw, err = base64.RawStdEncoding.DecodeString(param)
		if err != nil {
			return "", "", err
		}
	}
	for i := 0; i < len(raw); i++ {
		if raw[i] == ':' {
			return string(raw[:i]), string(raw[i+1:]), nil
		}
	}
	return "", "", errMalformedBasic
}

// constantTimeString is exported nowhere; kept local to avoid confusion with
// token comparison semantics.
func strEqFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(lowerASCII(a)), []byte(lowerASCII(b))) == 1
}

func lowerASCII(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}
