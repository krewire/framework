// Package storage provides app-level key-value storage: a small context-aware
// contract with in-memory and filesystem backends, plus a Provider that binds
// the active backend into the app DI container.
//
// Keys are "/"-separated paths ("sessions/abc", "uploads/2026/x.bin"). The
// filesystem backend maps them onto files under its root; path traversal is
// rejected. Values are raw bytes; encode structs at the call site.
package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// ErrNotFound is returned by backends when a key exists check fails; reads
// also report ok=false so callers can treat missing values as normal.
var ErrNotFound = errors.New("storage: key not found")

// KV is the minimal app-storage contract shared by every backend.
type KV interface {
	// Get returns the value for key; ok is false when absent.
	Get(ctx context.Context, key string) (val []byte, ok bool, err error)
	// Put stores val under key, creating intermediate namespaces.
	Put(ctx context.Context, key string, val []byte) error
	// Delete removes key; deleting an absent key is not an error.
	Delete(ctx context.Context, key string) error
	// List returns keys under prefix, sorted ascending.
	List(ctx context.Context, prefix string) ([]string, error)
}

// MemoryKV is a goroutine-safe in-memory backend.
type MemoryKV struct {
	mu    sync.RWMutex
	items map[string][]byte
}

// NewMemory returns an empty in-memory store.
func NewMemory() *MemoryKV {
	return &MemoryKV{items: map[string][]byte{}}
}

func (m *MemoryKV) Get(_ context.Context, key string) ([]byte, bool, error) {
	key = cleanKey(key)
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.items[key]
	if !ok {
		return nil, false, nil
	}
	out := make([]byte, len(val))
	copy(out, val)
	return out, true, nil
}

func (m *MemoryKV) Put(_ context.Context, key string, val []byte) error {
	key = cleanKey(key)
	if key == "" {
		return errors.New("storage: empty key")
	}
	cp := make([]byte, len(val))
	copy(cp, val)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[key] = cp
	return nil
}

func (m *MemoryKV) Delete(_ context.Context, key string) error {
	key = cleanKey(key)
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, key)
	return nil
}

func (m *MemoryKV) List(_ context.Context, prefix string) ([]string, error) {
	prefix = cleanKey(prefix)
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []string
	for k := range m.items {
		if strings.HasPrefix(k, prefix) {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out, nil
}

// FileKV persists keys as files beneath root.
type FileKV struct {
	root string
}

// NewFile returns a filesystem-backed store rooted at dir, creating it.
func NewFile(dir string) (*FileKV, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("storage: create root %s: %w", dir, err)
	}
	return &FileKV{root: dir}, nil
}

func (f *FileKV) Get(_ context.Context, key string) ([]byte, bool, error) {
	p, err := f.path(key)
	if err != nil {
		return nil, false, err
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("storage: get %s: %w", key, err)
	}
	return b, true, nil
}

func (f *FileKV) Put(_ context.Context, key string, val []byte) error {
	p, err := f.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("storage: mkdir for %s: %w", key, err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".kiw-tmp-*")
	if err != nil {
		return fmt.Errorf("storage: temp for %s: %w", key, err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(val); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("storage: write %s: %w", key, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("storage: close %s: %w", key, err)
	}
	if err := os.Rename(tmpName, p); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("storage: rename %s: %w", key, err)
	}
	return nil
}

func (f *FileKV) Delete(_ context.Context, key string) error {
	p, err := f.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("storage: delete %s: %w", key, err)
	}
	return nil
}

func (f *FileKV) List(_ context.Context, prefix string) ([]string, error) {
	prefix = cleanKey(prefix)
	dir := f.root
	if prefix != "" {
		dir = filepath.Join(f.root, filepath.FromSlash(prefix))
	}
	var out []string
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(f.root, p)
		if err != nil {
			return nil
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("storage: list %s: %w", prefix, err)
	}
	sort.Strings(out)
	return out, nil
}

func (f *FileKV) path(key string) (string, error) {
	for _, seg := range strings.Split(strings.ReplaceAll(key, "\\", "/"), "/") {
		if seg == ".." {
			return "", fmt.Errorf("storage: unsafe key %q", key)
		}
	}
	key = cleanKey(key)
	if key == "" {
		return "", errors.New("storage: empty key")
	}
	return filepath.Join(f.root, filepath.FromSlash(key)), nil
}

func cleanKey(key string) string {
	return strings.Trim(path.Clean("/"+strings.ReplaceAll(key, "\\", "/")), "/")
}
