// Tests for KWF-L5H2F
package config

import (
	"context"
	"testing"
)

// Spec: KWF-L5H2F FRK-SVC-010 Scope: Unit
func TestMemoryCenter_GetSet(t *testing.T) {
	c := NewMemoryCenter()
	ctx := context.Background()
	if err := c.Set(ctx, "db.host", "localhost:5432"); err != nil {
		t.Fatal(err)
	}
	v, err := c.Get(ctx, "db.host")
	if err != nil {
		t.Fatal(err)
	}
	if v.Data != "localhost:5432" {
		t.Fatalf("got %q", v.Data)
	}
}

// Spec: KWF-L5H2F FRK-SVC-010 Scope: Unit
func TestMemoryCenter_GetNotFound(t *testing.T) {
	c := NewMemoryCenter()
	_, err := c.Get(context.Background(), "missing")
	if !IsNotFound(err) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// Spec: KWF-L5H2F FRK-SVC-011 Scope: Unit
func TestMemoryCenter_Watch(t *testing.T) {
	c := NewMemoryCenter()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, _, err := c.Watch(ctx, "db.")
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Set(ctx, "db.host", "localhost"); err != nil {
		t.Fatal(err)
	}
	select {
	case change := <-ch:
		if change.Key != "db.host" || change.Value.Data != "localhost" {
			t.Fatalf("unexpected change: %+v", change)
		}
	default:
		t.Fatal("expected watch notification")
	}
}
