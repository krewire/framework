// Tests for KWF-T4X9P
package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec: KWF-T4X9P FRK-WASM-001 Scope: Unit
func TestBuildWASM_EmptyEntry_ReturnsUsageError(t *testing.T) {
	_, err := BuildWASM(Config{Entry: "", OutDir: t.TempDir()})
	if err == nil {
		t.Fatal("expected error for empty entry")
	}
	if !strings.Contains(err.Error(), "exit 2") {
		t.Fatalf("expected usage error, got: %v", err)
	}
}

// Spec: KWF-T4X9P FRK-WASM-001 Scope: Unit
func TestBuildWASM_EmptyOutDir_ReturnsUsageError(t *testing.T) {
	_, err := BuildWASM(Config{Entry: "./...", OutDir: ""})
	if err == nil {
		t.Fatal("expected error for empty out dir")
	}
	if !strings.Contains(err.Error(), "exit 2") {
		t.Fatalf("expected usage error, got: %v", err)
	}
}

// Spec: KWF-T4X9P FRK-WASM-001 Scope: Unit
func TestBuildWASM_InvalidGoRoot_ReturnsUsageError(t *testing.T) {
	_, err := BuildWASM(Config{Entry: "./...", OutDir: t.TempDir(), GoRoot: "/nonexistent"})
	if err == nil {
		t.Fatal("expected error for invalid goroot")
	}
	if !strings.Contains(err.Error(), "go toolchain not found") {
		t.Fatalf("expected toolchain error, got: %v", err)
	}
}

// Spec: KWF-T4X9P FRK-WASM-003 Scope: Unit
func TestFileDigest_KnownContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := fileDigest(path)
	if err != nil {
		t.Fatal(err)
	}
	// sha256("hello") = 2cf24dba...
	if !strings.HasPrefix(got, "2cf24dba") {
		t.Fatalf("unexpected digest: %s", got)
	}
}

// Spec: KWF-T4X9P FRK-WASM-003 Scope: Unit
func TestFingerprint_ShortPrefix(t *testing.T) {
	got := Fingerprint("abcdef1234567890")
	if got != "abcdef12" {
		t.Fatalf("fingerprint = %q, want abcdef12", got)
	}
}

// Spec: KWF-T4X9P FRK-WASM-003 Scope: Unit
func TestFingerprint_TooShort_ReturnsAsIs(t *testing.T) {
	got := Fingerprint("abc")
	if got != "abc" {
		t.Fatalf("fingerprint = %q, want abc", got)
	}
}

// Spec: KWF-T4X9P FRK-WASM-004 Scope: Unit
func TestDiagnoseBuildFailure_WasmTargetNotSupported(t *testing.T) {
	err := diagnoseBuildFailure("/usr/local/go", "GOOS=js GOARCH=wasm not recognized", os.ErrNotExist)
	if !strings.Contains(err.Error(), "wasm target not supported") {
		t.Fatalf("expected wasm target error, got: %v", err)
	}
}

// Spec: KWF-T4X9P FRK-WASM-004 Scope: Unit
func TestDiagnoseBuildFailure_ModuleResolution(t *testing.T) {
	err := diagnoseBuildFailure("/usr/local/go", "missing go.sum entry for github.com/foo/bar", os.ErrNotExist)
	if !strings.Contains(err.Error(), "module resolution failed") {
		t.Fatalf("expected module error, got: %v", err)
	}
}
