package web_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/krewire/framework/web"
)

func TestRequestID(t *testing.T) {
	var capturedID string
	h := web.RequestID()(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		capturedID = web.GetRequestID(req)
		w.WriteHeader(http.StatusOK)
	}))

	// Case 1: Auto-generated ID
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if capturedID == "" {
		t.Fatal("expected request ID to be generated")
	}
	if rec.Header().Get(web.HeaderRequestID) != capturedID {
		t.Fatalf("expected header %s to match captured %s", rec.Header().Get(web.HeaderRequestID), capturedID)
	}

	// Case 2: Propagate existing ID
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set(web.HeaderRequestID, "custom-trace-123")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	if capturedID != "custom-trace-123" {
		t.Fatalf("expected custom-trace-123, got %s", capturedID)
	}
	if rec2.Header().Get(web.HeaderRequestID) != "custom-trace-123" {
		t.Fatalf("expected header to echo custom-trace-123, got %s", rec2.Header().Get(web.HeaderRequestID))
	}
}

func TestRecover(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	h := web.Recover(logger)(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		panic("boom unexpected database crash")
	}))

	req := httptest.NewRequest("GET", "/danger", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Internal Server Error") {
		t.Fatalf("expected Internal Server Error in response, got %s", body)
	}
	// Stack trace should NOT be leaked in response body
	if strings.Contains(body, "panic recovered") || strings.Contains(body, "boom unexpected") {
		t.Fatal("sensitive panic information leaked to client response body")
	}

	// But it SHOULD be in server log
	logOutput := buf.String()
	if !strings.Contains(logOutput, "boom unexpected database crash") {
		t.Fatalf("expected panic in log, got %s", logOutput)
	}
}

func TestAccessLog(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	h := web.AccessLog(web.WithAccessLogger(logger))(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok response"))
	}))

	req := httptest.NewRequest("POST", "/create", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "status=201") {
		t.Fatalf("expected status=201 in log, got %s", logOutput)
	}
	if !strings.Contains(logOutput, "path=/create") {
		t.Fatalf("expected path=/create in log, got %s", logOutput)
	}
}
