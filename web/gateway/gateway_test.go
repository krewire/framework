// Tests for KWF-L5H2F
package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krewire/framework/service/registry"
)

// Spec: KWF-L5H2F FRK-SVC-020 Scope: Unit
func TestGateway_NoRoute_Returns404(t *testing.T) {
	g := New()
	req := httptest.NewRequest("GET", "/missing", nil)
	w := httptest.NewRecorder()
	g.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// Spec: KWF-L5H2F FRK-SVC-020 Scope: Unit
func TestGateway_NoRegistry_Returns502(t *testing.T) {
	g := New()
	g.AddRoute(Route{Path: "/api", Service: "backend"})
	req := httptest.NewRequest("GET", "/api", nil)
	w := httptest.NewRecorder()
	g.ServeHTTP(w, req)
	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", w.Code)
	}
}

// Spec: KWF-L5H2F FRK-SVC-020 Scope: Unit
func TestGateway_WithEndpoint_Proxies(t *testing.T) {
	// Start a backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello from backend"))
	}))
	defer backend.Close()

	reg := registry.NewMemoryRegistry()
	ctx := context.Background()
	_ = reg.Register(ctx, registry.Service{ID: "be-1", Name: "backend", Address: backend.Listener.Addr().String()})

	g := New(WithRegistry(reg))
	g.AddRoute(Route{Path: "/api", Service: "backend"})

	req := httptest.NewRequest("GET", "/api", nil)
	w := httptest.NewRecorder()
	g.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// Spec: KWF-L5H2F FRK-SVC-022 Scope: Unit
func TestGateway_Reload_ReplacesRoutes(t *testing.T) {
	g := New()
	g.AddRoute(Route{Path: "/old", Service: "old-svc"})
	g.Reload([]Route{{Path: "/new", Service: "new-svc"}})

	req := httptest.NewRequest("GET", "/old", nil)
	w := httptest.NewRecorder()
	g.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatal("old route should not exist after reload")
	}
}

// Spec: KWF-L5H2F FRK-SVC-023 Scope: Unit
func TestWriteProblem_ContentType(t *testing.T) {
	w := httptest.NewRecorder()
	writeProblem(w, http.StatusBadGateway, "service unavailable")
	if ct := w.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("expected problem+json, got %q", ct)
	}
}
