// Tests for KWF-T4X9P
package mount

import (
	"strings"
	"testing"
)

// Spec: KWF-T4X9P FRK-WASM-040 Scope: Unit
func TestFRK_WASM_040_Wrap_EmitsMarkersPropsAndEscaping(t *testing.T) {
	got := Wrap("Counter", Load, `{"initial":0}`, `<p>hi</p>`)
	want := `<div data-kiw-mount="Counter" data-kiw-hydrate="load" data-kiw-props="{&#34;initial&#34;:0}"><p>hi</p></div>`
	if got != want {
		t.Fatalf("Wrap = %q, want %q", got, want)
	}

	noProps := Wrap("Badge", Visible, "", `<b>x</b>`)
	if strings.Contains(noProps, "data-kiw-props") {
		t.Fatalf("empty props must omit attribute: %q", noProps)
	}
	nullProps := Wrap("Badge", Idle, "null", "")
	if strings.Contains(nullProps, "data-kiw-props") {
		t.Fatalf("null props must omit attribute: %q", nullProps)
	}
}

// Spec: KWF-T4X9P FRK-WASM-040 Scope: Unit
func TestFRK_WASM_040_ParseHydrate_ValidAndUnknown(t *testing.T) {
	for _, s := range []string{"load", "idle", "visible"} {
		if _, err := ParseHydrate(s); err != nil {
			t.Fatalf("ParseHydrate(%q) = %v", s, err)
		}
	}
	if _, err := ParseHydrate("hover"); err == nil {
		t.Fatal("unknown hydrate must error")
	} else if !strings.Contains(err.Error(), "load|idle|visible") {
		t.Fatalf("error must list valid options: %v", err)
	}
}

// Spec: KWF-T4X9P FRK-WASM-041 Scope: Unit
func TestFRK_WASM_041_Scan_FindsMountsInOrderWithRoundtripProps(t *testing.T) {
	page := `<!doctype html><html><head><title>t</title></head><body>` +
		Wrap("A", Load, `{"n":1}`, `<p>a</p>`) +
		`<main>` + Wrap("B", Visible, ``, `<i>b</i>`) + `</main>` +
		Wrap("C", Idle, `{"deep":{"k":"<v>"}}`, Wrap("Inner", Load, `{}`, `<u>n</u>`)) +
		`</body></html>`

	got, err := Scan(page)
	if err != nil {
		t.Fatal(err)
	}
	wantNames := []string{"A", "B", "C", "Inner"}
	if len(got) != len(wantNames) {
		t.Fatalf("scanned %d mounts (%+v), want %d in document order", len(got), got, len(wantNames))
	}
	for i, n := range wantNames {
		if got[i].Name != n {
			t.Fatalf("mount[%d].Name = %q, want %q", i, got[i].Name, n)
		}
	}
	if got[0].Hydrate != Load || got[0].Props != `{"n":1}` {
		t.Fatalf("mount A = %+v", got[0])
	}
	if got[1].Hydrate != Visible || got[1].Props != "" {
		t.Fatalf("mount B = %+v", got[1])
	}
	if got[2].Props != `{"deep":{"k":"<v>"}}` {
		t.Fatalf("escaped props did not roundtrip: %+v", got[2])
	}
	if got[3].Hydrate != Load {
		t.Fatalf("missing hydrate must default to load: %+v", got[3])
	}
}

// Spec: KWF-T4X9P FRK-WASM-041 Scope: Unit
func TestFRK_WASM_041_Scan_NoMountsAndPartialMarkers(t *testing.T) {
	got, err := Scan(`<html><body><p>plain</p></body></html>`)
	if err != nil || len(got) != 0 {
		t.Fatalf("plain page: mounts=%v err=%v", got, err)
	}

	partial, err := Scan(`<div data-kiw-mount="">x</div>`)
	if err != nil || len(partial) != 0 {
		t.Fatalf("nameless marker must be skipped gracefully: %v %v", partial, err)
	}

	if _, err := Scan("<p><b>broken"); err != nil {
		t.Fatalf("parser must recover from broken html: %v", err)
	}
}
