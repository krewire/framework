// Tests for KWF-WEB-B2X7D
package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ftest "github.com/krewire/framework/test"
)

var testSecret = []byte("kiw-test-secret")

// Spec: KWF-WEB-B2X7D FRK-AUTH-010 Scope: Package
func TestFRK_AUTH_010_BasicAuth(t *testing.T) {
	verify := func(id, pass string) (*Identity, error) {
		if id == "alice" && pass == "wonder" {
			return &Identity{Subject: id, Roles: []string{"reader"}}, nil
		}
		return nil, nil
	}
	r := NewRouter()
	r.Use(BasicAuth("kiw", verify))
	var got *Identity
	r.Get("/me", func(w http.ResponseWriter, req *http.Request, p Params) {
		got = IdentityFrom(req.Context())
		w.WriteHeader(200)
	})

	// valid
	rec := httptest.NewRecorder()
	req := ftest.NewRequest(t, http.MethodGet, "/me", nil)
	req.SetBasicAuth("alice", "wonder")
	r.ServeHTTP(rec, req)
	ftest.EqualStatus(t, rec, http.StatusOK)
	if got == nil || got.Subject != "alice" || got.Method != "basic" || !got.HasRole("reader") {
		t.Errorf("identity = %+v", got)
	}

	// bad password -> 401 + challenge
	rec2 := httptest.NewRecorder()
	req2 := ftest.NewRequest(t, http.MethodGet, "/me", nil)
	req2.SetBasicAuth("alice", "wrong")
	r.ServeHTTP(rec2, req2)
	ftest.EqualStatus(t, rec2, http.StatusUnauthorized)
	if ch := rec2.Header().Get("WWW-Authenticate"); !strings.HasPrefix(ch, `Basic realm="kiw"`) {
		t.Errorf("challenge = %q", ch)
	}

	// malformed header -> 401 without verifier (verifier would reject alice/wrong anyway;
	// use header that cannot decode)
	rec3 := httptest.NewRecorder()
	req3 := ftest.NewRequest(t, http.MethodGet, "/me", nil)
	req3.Header.Set("Authorization", "Basic !!!not-base64!!!")
	r.ServeHTTP(rec3, req3)
	ftest.EqualStatus(t, rec3, http.StatusUnauthorized)

	// anonymous -> 401
	rec4 := httptest.NewRecorder()
	r.ServeHTTP(rec4, ftest.NewRequest(t, http.MethodGet, "/me", nil))
	ftest.EqualStatus(t, rec4, http.StatusUnauthorized)
}

func sign(sub string, ttl time.Duration, extra Claims) string {
	cl := DefaultClaims(sub, ttl)
	for k, v := range extra {
		cl[k] = v
	}
	tok, err := SignJWT(testSecret, cl)
	if err != nil {
		panic(err)
	}
	return tok
}

// Spec: KWF-WEB-B2X7D FRK-AUTH-020+021 Scope: Package
func TestFRK_AUTH_020_JWT_SignParseAndMiddleware(t *testing.T) {
	// round trip
	claims, err := ParseJWT(testSecret, sign("bob", time.Minute, Claims{"roles": []string{"admin"}}))
	if err != nil {
		t.Fatal(err)
	}
	if claims["sub"] != "bob" {
		t.Errorf("sub = %v", claims["sub"])
	}

	r := NewRouter()
	r.Use(JWTAuth(testSecret))
	var id *Identity
	r.Get("/me", func(w http.ResponseWriter, req *http.Request, p Params) {
		id = IdentityFrom(req.Context())
		w.WriteHeader(200)
	})

	doAuth := func(token string) *httptest.ResponseRecorder {
		req := ftest.NewRequest(t, http.MethodGet, "/me", nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	// happy path with roles
	rec := doAuth(sign("bob", time.Minute, Claims{"roles": []string{"admin", "dev"}}))
	ftest.EqualStatus(t, rec, http.StatusOK)
	if id == nil || id.Subject != "bob" || len(id.Roles) != 2 || !id.HasRole("admin") {
		t.Errorf("identity = %+v", id)
	}
	if id.Claims["exp"] == nil {
		t.Error("claims not surfaced")
	}

	// missing token -> 401
	ftest.EqualStatus(t, doAuth(""), http.StatusUnauthorized)

	// expired -> 401
	ftest.EqualStatus(t, doAuth(sign("bob", -time.Minute, nil)), http.StatusUnauthorized)

	// tampered payload -> 401
	good := sign("bob", time.Minute, nil)
	parts := strings.Split(good, ".")
	parts[1] += "x"
	ftest.EqualStatus(t, doAuth(strings.Join(parts, ".")), http.StatusUnauthorized)

	// alg none swap -> 401
	noneHeader, _ := b64JSON(map[string]any{"alg": "none", "typ": "JWT"})
	payload := parts[1]
	forged := noneHeader + "." + payload + "."
	ftest.EqualStatus(t, doAuth(forged), http.StatusUnauthorized)

	// wrong secret -> 401
	otherTok, _ := SignJWT([]byte("other"), DefaultClaims("bob", time.Minute))
	ftest.EqualStatus(t, doAuth(otherTok), http.StatusUnauthorized)
}

// Spec: KWF-WEB-B2X7D FRK-AUTH-030+031 Scope: Package
func TestFRK_AUTH_030_PolicyGates_Before(t *testing.T) {
	jwtMW := JWTAuth(testSecret, func(o *JWTOptions) { o.ContinueOnMissing = true })
	adminGate := Require(Authenticated(), WithRoles("admin"))

	r := NewRouter()
	admin := r.Group("/admin", jwtMW, adminGate)
	admin.Get("/dash", func(w http.ResponseWriter, req *http.Request, p Params) { w.Write([]byte("secret")) })

	pub := r.Group("/public", jwtMW)
	pub.Get("/hello", func(w http.ResponseWriter, req *http.Request, p Params) {
		if IdentityFrom(req.Context()) == nil {
			w.Write([]byte("anon"))
			return
		}
		w.Write([]byte("user"))
	})

	// anonymous on /admin -> 401
	c1 := httptest.NewRecorder()
	r.ServeHTTP(c1, ftest.NewRequest(t, http.MethodGet, "/admin/dash", nil))
	ftest.EqualStatus(t, c1, http.StatusUnauthorized)

	// non-admin -> 403
	c2 := httptest.NewRecorder()
	req2 := ftest.NewRequest(t, http.MethodGet, "/admin/dash", nil)
	req2.Header.Set("Authorization", "Bearer "+sign("eve", time.Minute, Claims{"roles": []string{"dev"}}))
	r.ServeHTTP(c2, req2)
	ftest.EqualStatus(t, c2, http.StatusForbidden)

	// admin -> 200 secret
	c3 := httptest.NewRecorder()
	req3 := ftest.NewRequest(t, http.MethodGet, "/admin/dash", nil)
	req3.Header.Set("Authorization", "Bearer "+sign("al", time.Minute, Claims{"role": "admin"}))
	r.ServeHTTP(c3, req3)
	ftest.EqualStatus(t, c3, http.StatusOK)
	if c3.Body.String() != "secret" {
		t.Errorf("body = %q", c3.Body.String())
	}

	// public optional-auth: anon passes gate-free route
	c4 := httptest.NewRecorder()
	r.ServeHTTP(c4, ftest.NewRequest(t, http.MethodGet, "/public/hello", nil))
	ftest.EqualStatus(t, c4, http.StatusOK)
	if c4.Body.String() != "anon" {
		t.Errorf("public body = %q", c4.Body.String())
	}
}

// Spec: KWF-WEB-B2X7D FRK-AUTH-032 Scope: Package
func TestFRK_AUTH_032_AfterRequest_Observer(t *testing.T) {
	var seenStatus int
	var seenPath string
	r := NewRouter()
	r.Use(AfterRequest(func(req *http.Request, status int) {
		seenStatus = status
		seenPath = req.URL.Path
	}))
	r.Get("/ok", func(w http.ResponseWriter, req *http.Request, p Params) { w.WriteHeader(http.StatusTeapot) })

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, ftest.NewRequest(t, http.MethodGet, "/ok", nil))
	if seenStatus != http.StatusTeapot || seenPath != "/ok" {
		t.Errorf("after hook saw %d %q", seenStatus, seenPath)
	}
}

// Spec: KWF-WEB-B2X7D FRK-AUTH-033 Scope: Package
func TestFRK_AUTH_033_PolicySet_Registry(t *testing.T) {
	gates := PolicySet{
		"reader": WithRoles("reader"),
	}
	r := NewRouter()
	r.Use(JWTAuth(testSecret))
	lib := r.Group("/lib", gates.Require("reader"))
	lib.Get("/books", func(w http.ResponseWriter, req *http.Request, p Params) { w.Write([]byte("books")) })

	c := httptest.NewRecorder()
	req := ftest.NewRequest(t, http.MethodGet, "/lib/books", nil)
	req.Header.Set("Authorization", "Bearer "+sign("a", time.Minute, Claims{"roles": []string{"reader"}}))
	r.ServeHTTP(c, req)
	ftest.EqualStatus(t, c, http.StatusOK)

	c2 := httptest.NewRecorder()
	req2 := ftest.NewRequest(t, http.MethodGet, "/lib/books", nil)
	req2.Header.Set("Authorization", "Bearer "+sign("b", time.Minute, Claims{"roles": []string{"writer"}}))
	r.ServeHTTP(c2, req2)
	ftest.EqualStatus(t, c2, http.StatusForbidden)
}
