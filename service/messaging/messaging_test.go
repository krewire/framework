// Tests for KWF-L5H2F
package messaging

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// Spec: KWF-L5H2F FRK-SVC-050 Scope: Unit
func TestMemoryStream_PublishAndSubscribe(t *testing.T) {
	s := NewMemoryStream()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var received atomic.Int32
	_, err := s.Subscribe(ctx, "orders", func(ctx context.Context, msg Message) error {
		received.Add(1)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// small delay to let subscription start
	time.Sleep(10 * time.Millisecond)
	if err := s.Publish(ctx, "orders", []byte("order-1")); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if received.Load() < 1 {
		t.Fatalf("expected message received, got %d", received.Load())
	}
}

// Spec: KWF-L5H2F FRK-SVC-050 Scope: Unit
func TestMemoryStream_QueueSubscribe(t *testing.T) {
	s := NewMemoryStream()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var received atomic.Int32
	_, err := s.QueueSubscribe(ctx, "events", "group-a", func(ctx context.Context, msg Message) error {
		received.Add(1)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := s.Publish(ctx, "events", []byte("evt-1")); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if received.Load() < 1 {
		t.Fatalf("expected queue message received, got %d", received.Load())
	}
}

// Spec: KWF-L5H2F FRK-SVC-052 Scope: Unit
func TestMemoryStream_RedeliveryOnError(t *testing.T) {
	s := NewMemoryStream()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var attempts atomic.Int32
	_, err := s.Subscribe(ctx, "retry-topic", func(ctx context.Context, msg Message) error {
		n := attempts.Add(1)
		if n < 2 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := s.Publish(ctx, "retry-topic", []byte("msg")); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	if attempts.Load() < 2 {
		t.Fatalf("expected redelivery, got %d attempts", attempts.Load())
	}
}
