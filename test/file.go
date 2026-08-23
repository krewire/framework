package test

import (
	"os"
	"path/filepath"
	"testing"
)

func TempDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

func ReadFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", path, err)
	}
	return string(b)
}

func AssertFile(t *testing.T, path, want string) {
	t.Helper()
	got := ReadFile(t, path)
	if got != want {
		t.Errorf("AssertFile %s: got %q want %q", path, got, want)
	}
}

func WriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("WriteFile Mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", path, err)
	}
}
