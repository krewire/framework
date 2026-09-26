// Tests for KWF-WEB-R9T4C CORS middleware
package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	ftest "github.com/krewire/framework/test"
)

func TestWebCORS_Integration(t *testing.T) {
	r := NewRouter()
	r.Use(CORS(
		WithOrigins("https://frontend.dev"),
		WithMethods(http.MethodGet, http.MethodPost),
		WithHeaders("Authorization", "Content-Type"),
		WithCredentials(true),
		WithExposeHeaders("X-Total-Count"),
	))

	r.Get("/api/profile", func(w http.ResponseWriter, req *http.Request, p Params) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"user":"alice"}`))
	})

	// 1. Test Preflight OPTIONS
	preReq := ftest.NewRequest(t, http.MethodOptions, "/api/profile", nil)
	preReq.Header.Set("Origin", "https://frontend.dev")
	preReq.Header.Set("Access-Control-Request-Method", "GET")
	preRec := httptest.NewRecorder()

	r.ServeHTTP(preRec, preReq)

	ftest.Equal(t, http.StatusNoContent, preRec.Code)
	ftest.Equal(t, "https://frontend.dev", preRec.Header().Get("Access-Control-Allow-Origin"))
	ftest.Equal(t, "true", preRec.Header().Get("Access-Control-Allow-Credentials"))
	ftest.Equal(t, "GET, POST", preRec.Header().Get("Access-Control-Allow-Methods"))
	ftest.Equal(t, "X-Total-Count", preRec.Header().Get("Access-Control-Expose-Headers"))

	// 2. Test Actual GET request
	getReq := ftest.NewRequest(t, http.MethodGet, "/api/profile", nil)
	getReq.Header.Set("Origin", "https://frontend.dev")
	getRec := httptest.NewRecorder()

	r.ServeHTTP(getRec, getReq)

	ftest.Equal(t, http.StatusOK, getRec.Code)
	ftest.Equal(t, "https://frontend.dev", getRec.Header().Get("Access-Control-Allow-Origin"))
	ftest.Equal(t, `{"user":"alice"}`, getRec.Body.String())
}

func TestWebCORS_WildcardDefault(t *testing.T) {
	r := NewRouter()
	r.Use(CORS())
	r.Get("/api/data", func(w http.ResponseWriter, req *http.Request, p Params) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	req := ftest.NewRequest(t, http.MethodGet, "/api/data", nil)
	req.Header.Set("Origin", "https://anywhere.io")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	ftest.Equal(t, http.StatusOK, rec.Code)
	ftest.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
}
