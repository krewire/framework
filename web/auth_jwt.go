package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Claims is a JWT claim set.
type Claims map[string]any

// ErrInvalidToken classifies any rejected JWT (signature, format, alg, exp).
var ErrInvalidToken = errors.New("web: invalid token")

const jwtAlgHS256 = "HS256"

// SignJWT produces a compact HS256 JWS. Claims are emitted as-is; add
// "exp"/"sub" explicitly or via DefaultClaims.
func SignJWT(secret []byte, claims Claims) (string, error) {
	if len(secret) == 0 {
		return "", errors.New("web: empty jwt secret")
	}
	header := map[string]any{"alg": jwtAlgHS256, "typ": "JWT"}
	h, err := b64JSON(header)
	if err != nil {
		return "", err
	}
	p, err := b64JSON(claims)
	if err != nil {
		return "", err
	}
	signingInput := h + "." + p
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signingInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return signingInput + "." + sig, nil
}

// DefaultClaims seeds standard claims: sub, iat now, exp now+ttl.
func DefaultClaims(sub string, ttl time.Duration) Claims {
	now := time.Now().Unix()
	return Claims{"sub": sub, "iat": now, "exp": now + int64(ttl.Seconds())}
}

// ParseJWT verifies signature (alg pinned to HS256) and expiry, returning the
// parsed claims on success.
func ParseJWT(secret []byte, token string) (Claims, error) {
	if len(secret) == 0 {
		return nil, errors.New("web: empty jwt secret")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("%w: wrong segment count", ErrInvalidToken)
	}
	var hdr struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	rawHdr, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("%w: header", ErrInvalidToken)
	}
	if err := json.Unmarshal(rawHdr, &hdr); err != nil || hdr.Alg != jwtAlgHS256 {
		return nil, fmt.Errorf("%w: alg", ErrInvalidToken)
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(parts[0] + "." + parts[1]))
	want := mac.Sum(nil)
	got, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(want, got) {
		return nil, fmt.Errorf("%w: signature", ErrInvalidToken)
	}

	rawPayload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("%w: payload", ErrInvalidToken)
	}
	var claims Claims
	if err := json.Unmarshal(rawPayload, &claims); err != nil {
		return nil, fmt.Errorf("%w: payload json", ErrInvalidToken)
	}
	if expv, ok := claims["exp"]; ok {
		if exp, ok := toInt64(expv); !ok || time.Now().Unix() >= exp {
			return nil, fmt.Errorf("%w: expired", ErrInvalidToken)
		}
	}
	return claims, nil
}

// JWTOptions tunes the JWTAuth middleware.
type JWTOptions struct {
	// CookieName additionally accepts the token from this cookie.
	CookieName string
	// ContinueOnMissing lets requests without any token pass anonymously
	// (invalid supplied tokens still 401).
	ContinueOnMissing bool
	// Required enforces claim key==value pairs after verification.
	Required []ClaimCheck
}

// ClaimCheck asserts one claim equality.
type ClaimCheck struct {
	Key   string
	Value any
}

// JWTAuth verifies bearer credentials and stores the identity
// (Method "jwt") for downstream handlers/policies.
func JWTAuth(secret []byte, opts ...func(*JWTOptions)) Middleware {
	o := &JWTOptions{}
	for _, f := range opts {
		f(o)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := ""
			scheme, param, ok := authParam(r.Header.Get("Authorization"))
			if ok && strEqFold(scheme, "Bearer") {
				token = param
			} else if o.CookieName != "" {
				token = cookieVal(r, o.CookieName)
			}
			if token == "" {
				if o.ContinueOnMissing {
					next.ServeHTTP(w, r)
					return
				}
				Error(w, Unauthorized("missing bearer token"))
				return
			}
			claims, err := ParseJWT(secret, token)
			if err != nil {
				Error(w, Unauthorized("invalid token"))
				return
			}
			for _, rc := range o.Required {
				if v, ok := claims[rc.Key]; !ok || fmt.Sprint(v) != fmt.Sprint(rc.Value) {
					Error(w, Unauthorized("claim check failed"))
					return
				}
			}
			id := &Identity{Method: "jwt", Claims: claims}
			if sub, ok := claims["sub"]; ok {
				id.Subject = fmt.Sprint(sub)
			}
			switch rv := claims["roles"].(type) {
			case []any:
				for _, x := range rv {
					id.Roles = append(id.Roles, fmt.Sprint(x))
				}
			case string:
				id.Roles = append(id.Roles, rv)
			}
			if role, ok := claims["role"].(string); ok && len(id.Roles) == 0 {
				id.Roles = append(id.Roles, role)
			}
			next.ServeHTTP(w, r.WithContext(withIdentity(r.Context(), id)))
		})
	}
}

func b64JSON(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int64:
		return n, true
	case int:
		return int64(n), true
	case json.Number:
		i, err := n.Int64()
		return i, err == nil
	default:
		return 0, false
	}
}
