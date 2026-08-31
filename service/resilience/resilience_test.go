// Tests for KWF-L5H2F
package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Spec: KWF-L5H2F FRK-SVC-030 Scope: Unit
func TestCircuitBreaker_ClosedToOpen(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{Name: "test", Threshold: 2, Timeout: 100 * time.Millisecond})
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		_ = cb.Do(ctx, func() error { return errors.New("fail") })
	}
	if cb.State() != "open" {
		t.Fatalf("expected open, got %s", cb.State())
	}
	err := cb.Do(ctx, func() error { return nil })
	if err == nil {
		t.Fatal("expected ErrCircuitOpen")
	}
}

// Spec: KWF-L5H2F FRK-SVC-030 Scope: Unit
func TestCircuitBreaker_OpenToHalfOpenToClosed(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{Name: "test", Threshold: 1, Timeout: 50 * time.Millisecond, HalfOpenMax: 1})
	ctx := context.Background()
	_ = cb.Do(ctx, func() error { return errors.New("fail") })
	if cb.State() != "open" {
		t.Fatal("expected open")
	}
	time.Sleep(100 * time.Millisecond)
	err := cb.Do(ctx, func() error { return nil })
	if err != nil {
		t.Fatalf("half-open should allow: %v", err)
	}
	if cb.State() != "closed" {
		t.Fatalf("expected closed, got %s", cb.State())
	}
}

// Spec: KWF-L5H2F FRK-SVC-031 Scope: Unit
func TestRetryPolicy_SuccessOnRetry(t *testing.T) {
	p := RetryPolicy{MaxAttempts: 3, Backoff: 1 * time.Millisecond, MaxBackoff: 10 * time.Millisecond}
	attempts := 0
	err := p.Do(context.Background(), func() error {
		attempts++
		if attempts < 2 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

// Spec: KWF-L5H2F FRK-SVC-031 Scope: Unit
func TestRetryPolicy_Exhausted(t *testing.T) {
	p := RetryPolicy{MaxAttempts: 2, Backoff: 1 * time.Millisecond}
	err := p.Do(context.Background(), func() error { return errors.New("permanent") })
	if err == nil {
		t.Fatal("expected error after exhaustion")
	}
}

// Spec: KWF-L5H2F FRK-SVC-033 Scope: Unit
func TestBulkhead_Limit(t *testing.T) {
	b := NewBulkhead(1)
	ctx := context.Background()
	// First call should succeed
	if err := b.Do(ctx, func() error { return nil }); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
