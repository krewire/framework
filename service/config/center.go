// Package config provides a distributed configuration center (KWF-L5H2F
// FRK-SVC-010/011/012/013). It supports Get/Set/Watch operations with
// backends for etcd, Consul, S3, Git, and local/file. Hot reload pushes
// changes to registered callbacks without process restart.
package config

import (
	"context"
	"sync"
	"time"
)

// Value is a configuration value with metadata (FRK-SVC-010).
type Value struct {
	Key       string    `json:"key"`
	Data      string    `json:"data"`
	Version   int64     `json:"version"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Change is a configuration change event (FRK-SVC-010).
type Change struct {
	Key     string
	Value   Value
	Deleted bool
}

// Cancel stops a Watch goroutine (FRK-SVC-010).
type Cancel func()

// Center is the distributed config contract (FRK-SVC-010).
type Center interface {
	Get(ctx context.Context, key string) (Value, error)
	Set(ctx context.Context, key, value string) error
	Watch(ctx context.Context, prefix string) (<-chan Change, Cancel, error)
}

// ErrNotFound signals a key does not exist (FRK-SVC-010).
type ErrNotFound struct{ Key string }

func (e ErrNotFound) Error() string { return "config: key " + e.Key + " not found" }

// IsNotFound reports whether err is ErrNotFound.
func IsNotFound(err error) bool {
	_, ok := err.(ErrNotFound)
	return ok
}

// Backend is a config storage backend (FRK-SVC-012).
type Backend interface {
	Load() (map[string]Value, error)
	Save(key string, v Value) error
	Delete(key string) error
}

// MemoryBackend is an in-process backend for tests and local dev (FRK-SVC-012).
type MemoryBackend struct {
	mu   sync.RWMutex
	data map[string]Value
}

// NewMemoryBackend creates an empty in-memory backend.
func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{data: map[string]Value{}}
}

// Load returns all values.
func (b *MemoryBackend) Load() (map[string]Value, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make(map[string]Value, len(b.data))
	for k, v := range b.data {
		out[k] = v
	}
	return out, nil
}

// Save stores a value.
func (b *MemoryBackend) Save(key string, v Value) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.data[key] = v
	return nil
}

// Delete removes a value.
func (b *MemoryBackend) Delete(key string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.data, key)
	return nil
}

// MemoryCenter is an in-process config center for tests and local dev (FRK-SVC-010).
type MemoryCenter struct {
	mu       sync.RWMutex
	backend  Backend
	watchers map[string][]chan Change
	seq      int64
}

// NewMemoryCenter creates a config center backed by an in-memory backend.
func NewMemoryCenter() *MemoryCenter {
	return &MemoryCenter{
		backend:  NewMemoryBackend(),
		watchers: map[string][]chan Change{},
	}
}

// Get retrieves a value by key (FRK-SVC-010).
func (c *MemoryCenter) Get(ctx context.Context, key string) (Value, error) {
	data, err := c.backend.Load()
	if err != nil {
		return Value{}, err
	}
	v, ok := data[key]
	if !ok {
		return Value{}, ErrNotFound{Key: key}
	}
	return v, nil
}

// Set stores a value and notifies watchers (FRK-SVC-010/011).
func (c *MemoryCenter) Set(ctx context.Context, key, data string) error {
	c.mu.Lock()
	c.seq++
	v := Value{Key: key, Data: data, Version: c.seq, UpdatedAt: time.Now()}
	if err := c.backend.Save(key, v); err != nil {
		c.mu.Unlock()
		return err
	}
	c.notifyLocked(key, Change{Key: key, Value: v})
	c.mu.Unlock()
	return nil
}

// Watch streams changes for keys matching prefix (FRK-SVC-010/011).
func (c *MemoryCenter) Watch(ctx context.Context, prefix string) (<-chan Change, Cancel, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ch := make(chan Change, 16)
	c.watchers[prefix] = append(c.watchers[prefix], ch)
	canceled := false
	cancel := func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		if canceled {
			return
		}
		canceled = true
		close(ch)
	}
	return ch, cancel, nil
}

func (c *MemoryCenter) notifyLocked(key string, ch Change) {
	for prefix, watchers := range c.watchers {
		if len(prefix) == 0 || len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			for _, w := range watchers {
				select {
				case w <- ch:
				default:
				}
			}
		}
	}
}
