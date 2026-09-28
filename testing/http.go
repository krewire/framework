package test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func NewRequest(t *testing.T, method, target string, body io.Reader) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, target, body)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	return req
}

func Record(handler http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func EqualStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Errorf("EqualStatus: got %d want %d body %q", rec.Code, want, rec.Body.String())
	}
}
