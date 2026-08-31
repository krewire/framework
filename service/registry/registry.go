// Package registry provides service discovery (KWF-L5H2F FRK-SVC-001/002/003/004).
// A Registry registers services and resolves endpoints by name. Backends
// (Consul, etcd, NATS, DNS) implement the interface. Watch streams endpoint
// changes with backoff on transient failures.
package registry

import (
	"context"
	"sync"
	"time"
)

// Endpoint is a resolved service instance (FRK-SVC-001).
type Endpoint struct {
	ID      string            `json:"id"`
	Address string            `json:"address"`
	Meta    map[string]string `json:"meta,omitempty"`
}

// Service is the registration payload (FRK-SVC-002).
type Service struct {
	ID             string            `json:"id"`
	Name           string            `json:"name" validate:"required"`
	Address        string            `json:"address" validate:"required"`
	Meta           map[string]string `json:"meta,omitempty"`
	HealthCheckURL string            `json:"healthCheckUrl,omitempty"`
}

// Cancel stops a Watch goroutine and releases resources (FRK-SVC-001).
type Cancel func()

// Registry is the service discovery contract (FRK-SVC-001).
type Registry interface {
	Register(ctx context.Context, s Service) error
	Deregister(ctx context.Context, id string) error
	Discover(ctx context.Context, name string) ([]Endpoint, error)
	Watch(ctx context.Context, name string) (<-chan []Endpoint, Cancel, error)
}

// ErrNotFound signals no endpoints exist for a service name (FRK-SVC-001).
type ErrNotFound struct {
	Name string
}

func (e ErrNotFound) Error() string {
	return "service " + e.Name + " not found"
}

// IsNotFound reports whether err is ErrNotFound.
func IsNotFound(err error) bool {
	_, ok := err.(ErrNotFound)
	return ok
}

// Backoff controls retry timing for Watch failures (FRK-SVC-004).
type Backoff struct {
	Initial time.Duration
	Max     time.Duration
	Factor  float64
	jumps   int
}

// NewBackoff creates a backoff with sensible defaults (FRK-SVC-004).
func NewBackoff() *Backoff {
	return &Backoff{Initial: 100 * time.Millisecond, Max: 30 * time.Second, Factor: 2}
}

// Next returns the sleep duration for the current attempt and advances.
func (b *Backoff) Next() time.Duration {
	d := b.Initial
	for i := 0; i < b.jumps; i++ {
		d = time.Duration(float64(d) * b.Factor)
	}
	if d > b.Max {
		d = b.Max
	}
	b.jumps++
	return d
}

// Reset clears the attempt counter.
func (b *Backoff) Reset() { b.jumps = 0 }

// MemoryRegistry is an in-process backend for tests and local dev (FRK-SVC-003).
type MemoryRegistry struct {
	mu        sync.RWMutex
	services  map[string]Service
	endpoints map[string][]Endpoint
	watchers  map[string][]chan []Endpoint
}

// NewMemoryRegistry creates an empty in-memory registry.
func NewMemoryRegistry() *MemoryRegistry {
	return &MemoryRegistry{
		services:  map[string]Service{},
		endpoints: map[string][]Endpoint{},
		watchers:  map[string][]chan []Endpoint{},
	}
}

// Register adds a service and notifies watchers (FRK-SVC-001).
func (r *MemoryRegistry) Register(ctx context.Context, s Service) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.services[s.ID] = s
	ep := Endpoint{ID: s.ID, Address: s.Address, Meta: s.Meta}
	r.endpoints[s.Name] = append(r.endpoints[s.Name], ep)
	r.notifyLocked(s.Name, r.endpoints[s.Name])
	return nil
}

// Deregister removes a service and notifies watchers (FRK-SVC-001).
func (r *MemoryRegistry) Deregister(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.services[id]
	if !ok {
		return nil
	}
	delete(r.services, id)
	var remaining []Endpoint
	for _, ep := range r.endpoints[s.Name] {
		if ep.ID != id {
			remaining = append(remaining, ep)
		}
	}
	r.endpoints[s.Name] = remaining
	r.notifyLocked(s.Name, remaining)
	return nil
}

// Discover returns current endpoints for a service (FRK-SVC-001).
func (r *MemoryRegistry) Discover(ctx context.Context, name string) ([]Endpoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	eps, ok := r.endpoints[name]
	if !ok || len(eps) == 0 {
		return nil, ErrNotFound{Name: name}
	}
	out := make([]Endpoint, len(eps))
	copy(out, eps)
	return out, nil
}

// Watch streams endpoint changes (FRK-SVC-001/004).
func (r *MemoryRegistry) Watch(ctx context.Context, name string) (<-chan []Endpoint, Cancel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ch := make(chan []Endpoint, 1)
	r.watchers[name] = append(r.watchers[name], ch)
	canceled := false
	cancel := func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		if canceled {
			return
		}
		canceled = true
		close(ch)
	}
	return ch, cancel, nil
}

func (r *MemoryRegistry) notifyLocked(name string, eps []Endpoint) {
	for _, ch := range r.watchers[name] {
		select {
		case ch <- eps:
		default:
		}
	}
}
