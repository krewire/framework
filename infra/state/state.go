// Package state manages infrastructure state persistence and locking
// (KWF-B7N3D FRK-INFRA-020/021/022/023). State is JSON, stored locally or in a
// remote backend. Locking prevents concurrent mutations.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// State is the persisted infrastructure graph (FRK-INFRA-020).
type State struct {
	mu       sync.RWMutex
	backend  Backend
	locked   bool
	lockInfo LockInfo
}

// Backend persists state (FRK-INFRA-020).
type Backend interface {
	Load() ([]byte, error)
	Save(data []byte) error
	Lock(info LockInfo) error
	Unlock() error
}

// LockInfo identifies who holds a lock (FRK-INFRA-021/022).
type LockInfo struct {
	ID        string    `json:"id"`
	Who       string    `json:"who"`
	Created   time.Time `json:"created"`
	Operation string    `json:"operation"`
}

// ErrAlreadyLocked signals a concurrent operation holds the lock (FRK-INFRA-023).
type ErrAlreadyLocked struct {
	Owner LockInfo
}

func (e ErrAlreadyLocked) Error() string {
	return fmt.Sprintf("state locked by %s since %s (%s)", e.Owner.Who, e.Owner.Created.Format(time.RFC3339), e.Owner.Operation)
}

// New creates a state manager backed by backend (FRK-INFRA-020).
func New(backend Backend) *State {
	return &State{backend: backend}
}

// Load reads state from the backend (FRK-INFRA-020).
func (s *State) Load() (map[string]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := s.backend.Load()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("state: decode: %w", err)
	}
	return out, nil
}

// Save persists state to the backend without locking (FRK-INFRA-022).
func (s *State) Save(resources map[string]any) error {
	data, err := json.MarshalIndent(resources, "", "  ")
	if err != nil {
		return fmt.Errorf("state: encode: %w", err)
	}
	return s.backend.Save(data)
}

// Lock acquires the state lock if not already held (FRK-INFRA-021/023).
func (s *State) Lock(info LockInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.locked {
		return ErrAlreadyLocked{Owner: s.lockInfo}
	}
	if err := s.backend.Lock(info); err != nil {
		return err
	}
	s.locked = true
	s.lockInfo = info
	return nil
}

// Unlock releases the state lock (FRK-INFRA-021/023).
func (s *State) Unlock() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.locked {
		return nil
	}
	if err := s.backend.Unlock(); err != nil {
		return err
	}
	s.locked = false
	s.lockInfo = LockInfo{}
	return nil
}

// FileBackend stores state in a local JSON file (FRK-INFRA-020).
type FileBackend struct {
	Path string
	mu   sync.Mutex
}

// Load reads the state file.
func (b *FileBackend) Load() ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	data, err := os.ReadFile(b.Path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return data, err
}

// Save writes the state file atomically via temp+rename.
func (b *FileBackend) Save(data []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	dir := filepath.Dir(b.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("state: mkdir: %w", err)
	}
	tmp := b.Path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("state: write temp: %w", err)
	}
	return os.Rename(tmp, b.Path)
}

// Lock is a no-op for file backend (local dev).
func (b *FileBackend) Lock(info LockInfo) error { return nil }

// Unlock is a no-op for file backend (local dev).
func (b *FileBackend) Unlock() error { return nil }
