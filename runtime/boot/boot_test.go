// Tests for KWF-T4X9P
package boot

import (
	"strings"
	"testing"
)

// Spec: KWF-T4X9P FRK-WASM-041 Scope: Unit
func TestFRK_WASM_041_FlattenProps_ScalarsAndNull(t *testing.T) {
	got, err := FlattenProps([]byte(`{"label":"Count","initial":0,"ratio":1.5,"on":true,"empty":null}`))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"label": "Count", "initial": "0", "ratio": "1.5", "on": "true", "empty": ""}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("props[%q] = %q, want %q", k, got[k], v)
		}
	}
	if len(got) != len(want) {
		t.Errorf("unexpected extra props: %v", got)
	}

	if p, err := FlattenProps(nil); err != nil || len(p) != 0 {
		t.Errorf("nil payload = %v, %v", p, err)
	}
	if p, err := FlattenProps([]byte("  \n")); err != nil || len(p) != 0 {
		t.Errorf("blank payload = %v, %v", p, err)
	}
	if _, err := FlattenProps([]byte(`{bad`)); err == nil || !strings.Contains(err.Error(), "decode props") {
		t.Errorf("broken JSON must error with decode context: %v", err)
	}
}

// Spec: KWF-T4X9P FRK-WASM-041 Scope: Unit
func TestFRK_WASM_041_FlattenProps_NestedEncodedAsString(t *testing.T) {
	got, err := FlattenProps([]byte(`{"deep":{"k":[1,2]}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got["deep"] != `{"k":[1,2]}` {
		t.Fatalf("nested value = %q", got["deep"])
	}
}
