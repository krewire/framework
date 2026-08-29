// Tests for KWF-TEST-M4P9Q
package test

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// Spec: KWF-TEST-M4P9Q KWF-TST-M4P-010 Scope: Unit
func TestKWF_TST_M4P_010_Equal_Valid(t *testing.T) {
	Spec(t, "KWF-TEST-M4P9Q", "KWF-TST-M4P-010")
	// should not fail for equal
	Equal(t, "a", "a")
	Equal(t, 123, 123)
}

// Spec: KWF-TEST-M4P9Q KWF-TST-M4P-011 Scope: Unit
func TestKWF_TST_M4P_011_NoError_HasError(t *testing.T) {
	Spec(t, "KWF-TEST-M4P9Q", "KWF-TST-M4P-011")
	NoError(t, nil)
	HasError(t, os.ErrNotExist)
}

// Spec: KWF-TEST-M4P9Q KWF-TST-M4P-012 Scope: Unit
func TestKWF_TST_M4P_012_Contains_Valid(t *testing.T) {
	Spec(t, "KWF-TEST-M4P9Q", "KWF-TST-M4P-012")
	Contains(t, "<html>hello</html>", "hello")
	NotContains(t, "<html>hello</html>", "bad")
}

// Spec: KWF-TEST-M4P9Q KWF-TST-M4P-020 Scope: Unit
func TestKWF_TST_M4P_020_HTTP_Valid(t *testing.T) {
	Spec(t, "KWF-TEST-M4P9Q", "KWF-TST-M4P-020")
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	})
	req := NewRequest(t, "GET", "/", nil)
	rec := Record(handler, req)
	EqualStatus(t, rec, 200)
	Contains(t, rec.Body.String(), "ok")
}

// Spec: KWF-TEST-M4P9Q KWF-TST-M4P-031 Scope: Unit
func TestKWF_TST_M4P_031_File_Valid(t *testing.T) {
	Spec(t, "KWF-TEST-M4P9Q", "KWF-TST-M4P-031")
	dir := TempDir(t)
	path := filepath.Join(dir, "a/b.txt")
	WriteFile(t, path, "hello")
	AssertFile(t, path, "hello")
	ReadFile(t, path)
}

// Spec: KWF-TEST-M4P9Q KWF-TST-M4P-032 Scope: Unit
func TestKWF_TST_M4P_032_Golden_Valid(t *testing.T) {
	Spec(t, "KWF-TEST-M4P9Q", "KWF-TST-M4P-032")
	name := "test_golden_mvp"
	path := filepath.Join("testdata", name+".golden")
	_ = os.Remove(path)
	t.Setenv("UPDATE_GOLDEN", "1")
	Golden(t, name, "hello golden")
	t.Setenv("UPDATE_GOLDEN", "")
	Golden(t, name, "hello golden")
	_ = os.Remove(path)
}

// Spec: KWF-TEST-M4P9Q KWF-TST-M4P-040 Scope: Unit
func TestKWF_TST_M4P_040_Spec_Valid(t *testing.T) {
	Spec(t, "KWF-TEST-M4P9Q", "KWF-TST-M4P-040")
	Spec(t, "KWF-TEST-M4P9Q", "KWF-TST-M4P-040")
}
