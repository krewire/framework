// Tests for KWF-L5H2F
package registry

import (
	"context"
	"testing"
	"time"
)

// Spec: KWF-L5H2F FRK-SVC-001 Scope: Unit
func TestMemoryRegistry_RegisterAndDiscover(t *testing.T) {
	r := NewMemoryRegistry()
	ctx := context.Background()
	svc := Service{ID: "web-1", Name: "web", Address: "localhost:8080"}
	if err := r.Register(ctx, svc); err != nil {
		t.Fatal(err)
	}
	eps, err := r.Discover(ctx, "web")
	if err != nil {
		t.Fatal(err)
	}
	if len(eps) != 1 || eps[0].Address != "localhost:8080" {
		t.Fatalf("unexpected endpoints: %+v", eps)
	}
}

// Spec: KWF-L5H2F FRK-SVC-001 Scope: Unit
func TestMemoryRegistry_DiscoverNotFound(t *testing.T) {
	r := NewMemoryRegistry()
	_, err := r.Discover(context.Background(), "missing")
	if !IsNotFound(err) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// Spec: KWF-L5H2F FRK-SVC-001 Scope: Unit
func TestMemoryRegistry_Deregister(t *testing.T) {
	r := NewMemoryRegistry()
	ctx := context.Background()
	svc := Service{ID: "web-1", Name: "web", Address: "localhost:8080"}
	_ = r.Register(ctx, svc)
	if err := r.Deregister(ctx, "web-1"); err != nil {
		t.Fatal(err)
	}
	_, err := r.Discover(ctx, "web")
	if !IsNotFound(err) {
		t.Fatal("expected not found after deregister")
	}
}

// Spec: KWF-L5H2F FRK-SVC-004 Scope: Unit
func TestBackoff_NextAndReset(t *testing.T) {
	b := &Backoff{Initial: 100 * time.Millisecond, Max: 1 * time.Second, Factor: 2}
	d1 := b.Next()
	d2 := b.Next()
	if d2 < d1 {
		t.Fatalf("backoff should increase: %v -> %v", d1, d2)
	}
	b.Reset()
	d3 := b.Next()
	if d3 != d1 {
		t.Fatalf("reset should return to initial: got %v want %v", d3, d1)
	}
}
