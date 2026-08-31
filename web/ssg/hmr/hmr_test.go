// Tests for KWF-209JV
package hmr

import (
	"net/http/httptest"
	"testing"
	"time"
)

// Spec: KWF-209JV Scope: Unit
func TestNewServer(t *testing.T) {
	s := NewServer()
	if s == nil {
		t.Fatal("expected non-nil server")
	}
	if s.ClientCount() != 0 {
		t.Fatalf("expected 0 clients, got %d", s.ClientCount())
	}
}

// Spec: KWF-209JV Scope: Unit
func TestBroadcast_NoClients(t *testing.T) {
	s := NewServer()
	s.Broadcast(Event{Type: EventReload})
	if s.ClientCount() != 0 {
		t.Fatal("expected 0 clients")
	}
}

// Spec: KWF-209JV Scope: Unit
func TestServeHTTP_SSE(t *testing.T) {
	s := NewServer()
	req := httptest.NewRequest("GET", "/__krewire/hmr", nil)
	w := httptest.NewRecorder()

	// Use a channel to signal when handler is done
	done := make(chan struct{})
	go func() {
		s.ServeHTTP(w, req)
		close(done)
	}()

	// Give handler time to start
	time.Sleep(50 * time.Millisecond)

	if s.ClientCount() != 1 {
		t.Fatalf("expected 1 client, got %d", s.ClientCount())
	}

	s.Broadcast(Event{Type: EventReload, Path: "/index.html"})

	// Cancel request to stop handler
	cancelReq := req.Clone(req.Context())
	_ = cancelReq

	time.Sleep(50 * time.Millisecond)
}

// Spec: KWF-209JV Scope: Unit
func TestEvent_Types(t *testing.T) {
	tests := []struct {
		ev   Event
		want string
	}{
		{Event{Type: EventReload}, "reload"},
		{Event{Type: EventStyleUpdate}, "style-update"},
		{Event{Type: EventContentUpdate}, "content-update"},
	}
	for _, tt := range tests {
		if string(tt.ev.Type) != tt.want {
			t.Fatalf("event type = %q, want %q", tt.ev.Type, tt.want)
		}
	}
}

// Spec: KWF-209JV Scope: Unit
func TestEventLog_Truncation(t *testing.T) {
	s := NewServer()
	for i := 0; i < 100; i++ {
		s.Broadcast(Event{Type: EventReload})
	}
	s.mu.RLock()
	count := len(s.eventLog)
	s.mu.RUnlock()
	if count > 64 {
		t.Fatalf("event log should be truncated to 64, got %d", count)
	}
}
