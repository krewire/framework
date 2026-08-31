// Package hmr provides live reload and hot module replacement for the SSG
// dev server (KWF-209JV). It serves a WebSocket endpoint at /__krewire/hmr
// that broadcasts reload/style-update/content-update events to connected
// browsers when incremental builds detect changes.
package hmr

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// EventType is the kind of HMR event (KWF-209JV).
type EventType string

const (
	// EventReload signals a full page reload.
	EventReload EventType = "reload"
	// EventStyleUpdate signals a CSS-only update (no reload needed).
	EventStyleUpdate EventType = "style-update"
	// EventContentUpdate signals a content change with path.
	EventContentUpdate EventType = "content-update"
)

// Event is an HMR message sent to connected clients (KWF-209JV).
type Event struct {
	Type    EventType `json:"type"`
	Path    string    `json:"path,omitempty"`
	Message string    `json:"message,omitempty"`
}

// Server is an HMR WebSocket endpoint (KWF-209JV).
type Server struct {
	mu       sync.RWMutex
	clients  map[chan Event]struct{}
	eventLog []Event
}

// NewServer creates an HMR server (KWF-209JV).
func NewServer() *Server {
	return &Server{
		clients:  map[chan Event]struct{}{},
		eventLog: make([]Event, 0, 64),
	}
}

// ServeHTTP handles WebSocket upgrade requests (KWF-209JV).
// For simplicity, this uses Server-Sent Events (SSE) instead of raw
// WebSockets — SSE is sufficient for one-directional HMR broadcasts
// and requires no external dependency.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	ch := make(chan Event, 16)
	s.mu.Lock()
	s.clients[ch] = struct{}{}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, ch)
		s.mu.Unlock()
		close(ch)
	}()

	// Send recent event log to catch up a reconnecting client
	s.mu.RLock()
	for _, ev := range s.eventLog {
		s.writeEvent(w, ev, flusher)
	}
	s.mu.RUnlock()

	// Heartbeat to keep connection alive
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return
			}
			s.writeEvent(w, ev, flusher)
		case <-ticker.C:
			fmt.Fprintf(w, ":\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) writeEvent(w http.ResponseWriter, ev Event, flusher http.Flusher) {
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\n", ev.Type)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

// Broadcast sends an event to all connected clients (KWF-209JV).
func (s *Server) Broadcast(ev Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eventLog = append(s.eventLog, ev)
	if len(s.eventLog) > 64 {
		s.eventLog = s.eventLog[len(s.eventLog)-64:]
	}
	for ch := range s.clients {
		select {
		case ch <- ev:
		default:
		}
	}
	slog.Debug("hmr broadcast", "type", ev.Type, "path", ev.Path)
}

// ClientCount returns the number of connected clients (for observability).
func (s *Server) ClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}
