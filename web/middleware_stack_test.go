// Tests for KWL-P8W2N
package web

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Spec: KWL-P8W2N KWF-HTTPV-005 S1 Scope: Package
func TestKWL_HTTPV_005_RecoverMiddleware_LogsStackAndHidesInternals(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	mw := RecoverMiddleware(logger)
	handler := mw(http.HandlerFunc(panicMarkerHandler))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/panic", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if body := rec.Body.String(); strings.Contains(body, "kaboom-marker") {
		t.Errorf("response leaked panic internals: %q", body)
	}
	logged := buf.String()
	if !strings.Contains(logged, "panic recovered") || !strings.Contains(logged, "kaboom-marker") {
		t.Errorf("log missing panic record:\n%s", logged)
	}
	if !strings.Contains(logged, "panicMarkerHandler") || !strings.Contains(logged, "middleware_stack_test.go") {
		t.Errorf("log missing stack frames naming the handler:\n%s", logged)
	}
}

// Spec: KWL-P8W2N KWF-HTTPV-006 Scope: Package
func TestKWL_HTTPV_006_App_DefaultsRecoveryAndAccessLog(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	app := NewApp()
	app.Method(http.MethodGet, "/boom", func(w http.ResponseWriter, _ *http.Request, _ Params) {
		panic("app-boom")
	})
	srv := app.Handler()

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/boom", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 from default recovery", rec.Code)
	}

	rec2 := httptest.NewRecorder()
	srv.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/ok-route", nil))

	logged := buf.String()
	if !strings.Contains(logged, "panic recovered") || !strings.Contains(logged, "app-boom") {
		t.Errorf("default recovery did not log the panic:\n%s", logged)
	}
	if !strings.Contains(logged, "http request") {
		t.Errorf("default access log missing:\n%s", logged)
	}
}

func panicMarkerHandler(w http.ResponseWriter, _ *http.Request) {
	panic("kaboom-marker")
}

// Spec: KWL-P8W2N KWF-HTTPV-011 S5 Scope: Package
func TestKWF_HTTPV_011_RecoverResponse_CarriesCorrelationIdMirroredInLog(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	mw := RecoverMiddleware(logger)
	handler := mw(http.HandlerFunc(panicMarkerHandler))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/panic", nil))

	id := rec.Header().Get("X-Correlation-Id")
	if id == "" {
		t.Fatal("response missing X-Correlation-Id header")
	}
	if !strings.Contains(rec.Body.String(), id) {
		t.Errorf("body %q does not carry correlation id %q", rec.Body.String(), id)
	}
	if !strings.Contains(buf.String(), id) {
		t.Errorf("log line does not mirror the correlation id:\n%s", buf.String())
	}
}
