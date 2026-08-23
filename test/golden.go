package test

import (
	"os"
	"path/filepath"
	"testing"
)

func Golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("Golden Mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("Golden Write %s: %v", path, err)
		}
		return
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Golden %s: %v (run UPDATE_GOLDEN=1 to create)", name, err)
	}
	if string(b) != got {
		t.Errorf("Golden %s: got %q want %q", name, got, string(b))
	}
}
