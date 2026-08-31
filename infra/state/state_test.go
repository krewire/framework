// Tests for KWF-B7N3D
package state

import (
	"testing"
	"time"
)

// Spec: KWF-B7N3D FRK-INFRA-020 Scope: Unit
func TestFileBackend_LoadMissing(t *testing.T) {
	b := &FileBackend{Path: "/tmp/krewire-test-state-missing.json"}
	data, err := b.Load()
	if err != nil {
		t.Fatal(err)
	}
	if data != nil {
		t.Fatalf("expected nil, got %q", data)
	}
}

// Spec: KWF-B7N3D FRK-INFRA-020 Scope: Unit
func TestState_SaveAndLoad(t *testing.T) {
	path := t.TempDir() + "/state.json"
	s := New(&FileBackend{Path: path})

	resources := map[string]any{"web": map[string]any{"kind": "Compute"}}
	if err := s.Save(resources); err != nil {
		t.Fatal(err)
	}
	loaded, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded["web"] == nil {
		t.Fatal("expected web resource in loaded state")
	}
}

// Spec: KWF-B7N3D FRK-INFRA-021 Scope: Unit
func TestState_LockAndUnlock(t *testing.T) {
	path := t.TempDir() + "/state.json"
	s := New(&FileBackend{Path: path})

	info := LockInfo{ID: "op-1", Who: "test", Operation: "apply", Created: time.Now()}
	if err := s.Lock(info); err != nil {
		t.Fatal(err)
	}
	err := s.Lock(info)
	if err == nil {
		t.Fatal("expected lock conflict")
	}
	if _, ok := err.(ErrAlreadyLocked); !ok {
		t.Fatalf("expected ErrAlreadyLocked, got %T", err)
	}
	if err := s.Unlock(); err != nil {
		t.Fatal(err)
	}
	if err := s.Unlock(); err != nil {
		t.Fatal("unlock should be idempotent")
	}
}
