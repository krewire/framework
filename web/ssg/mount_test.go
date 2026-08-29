// Tests for KWF-T4X9P
package ssg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec: KWF-T4X9P FRK-WASM-040 Scope: Unit
func TestFRK_WASM_040_MountTemplateFuncEmitsMarkers(t *testing.T) {
	site, err := LoadFromDir("testdata/mount")
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if _, err := site.Build(out); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(body)

	for _, want := range []string{
		`data-kiw-mount="Counter"`,
		`data-kiw-hydrate="load"`,
		`data-kiw-props="{`,
		// SSR content stays fully rendered inside the mount wrapper.
		`<span class="count">0</span>`,
		`<button type="button">+1</button>`,
		// Scoped component CSS contract unchanged.
		`data-kiw-component="Counter"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("mount html missing %q\n got %q", want, html[:min(len(html), 800)])
		}
	}

	css, err := os.ReadFile(filepath.Join(out, "assets", "style.css"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(css), `[data-kiw-component="Counter"]`) {
		t.Errorf("scoped css missing Counter: %s", string(css))
	}
}

// Spec: KWF-T4X9P FRK-WASM-040 Scope: Unit
func TestFRK_WASM_040_MountRejectsUnknownHydrateValue(t *testing.T) {
	site, err := LoadFromDir("testdata/mount")
	if err != nil {
		t.Fatal(err)
	}
	if err := site.prepare(); err != nil {
		t.Fatalf("template preparation must succeed: %v", err)
	}
	if _, err := site.renderMount("Counter", "hover", nil); err == nil {
		t.Fatal("unknown hydrate value must error")
	} else if !strings.Contains(err.Error(), "load|idle|visible") {
		t.Fatalf("error must list valid hydrate values: %v", err)
	}
	if _, err := site.renderMount("Nope", "load", nil); err == nil {
		t.Fatal("undefined component in mount must error")
	}
}
