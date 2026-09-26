package web

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	ftest "github.com/krewire/framework/test"
)

func TestWebHealth_Integration(t *testing.T) {
	r := NewRouter()
	r.Use(Health(
		WithCheck("database", func(ctx context.Context) error { return nil }),
		WithCheck("kv", func(ctx context.Context) error { return nil }),
	))

	r.Get("/hello", func(w http.ResponseWriter, req *http.Request, p Params) {
		w.WriteHeader(http.StatusOK)
	})

	// 1. /healthz liveness
	req := ftest.NewRequest(t, http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	ftest.Equal(t, http.StatusOK, rec.Code)

	// 2. /readyz readiness
	reqReady := ftest.NewRequest(t, http.MethodGet, "/readyz", nil)
	recReady := httptest.NewRecorder()
	r.ServeHTTP(recReady, reqReady)
	ftest.Equal(t, http.StatusOK, recReady.Code)

	// 3. Normal route still works
	reqHello := ftest.NewRequest(t, http.MethodGet, "/hello", nil)
	recHello := httptest.NewRecorder()
	r.ServeHTTP(recHello, reqHello)
	ftest.Equal(t, http.StatusOK, recHello.Code)
}

func TestWebHealth_Degraded(t *testing.T) {
	r := NewRouter()
	r.Use(Health(
		WithCheck("database", func(ctx context.Context) error { return errors.New("connection refused") }),
	))

	req := ftest.NewRequest(t, http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	ftest.Equal(t, http.StatusServiceUnavailable, rec.Code)
}
