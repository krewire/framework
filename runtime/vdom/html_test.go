// Tests for KWF-T4X9P
package vdom

import (
	"strings"
	"testing"
)

// Spec: KWF-T4X9P FRK-WASM-022 Scope: Unit
func TestFRK_WASM_022_RenderHTML_SharesNormalizationWithDiff(t *testing.T) {
	boolBare := RenderHTML(El("input", map[string]string{"disabled": ""}))
	if !strings.Contains(boolBare, "<input disabled>") || strings.Contains(boolBare, `disabled=`) {
		t.Fatalf("boolean attr must render bare: %q", boolBare)
	}

	withDisabled := El("button", map[string]string{"disabled": ""}, Text("go"))
	withoutDisabled := El("button", map[string]string{}, Text("go"))
	if p := Diff(withDisabled, withoutDisabled); len(p) == 0 {
		t.Fatalf("removing a bool prop must be detectable")
	}

	spaced := El("p", map[string]string{"class": "  a   b  "})
	got := RenderHTML(spaced)
	if !strings.Contains(got, `class="a b"`) {
		t.Fatalf("class must collapse whitespace: %q", got)
	}
	retyped := El("p", map[string]string{"class": "a b"})
	if p := Diff(spaced, retyped); len(p) != 0 {
		t.Fatalf("spacing-only class change must not patch: %+v", p)
	}

	emptyNonBool := RenderHTML(El("p", map[string]string{"title": ""}))
	if strings.Contains(emptyNonBool, "title") {
		t.Fatalf("empty non-bool attr must be dropped: %q", emptyNonBool)
	}
}

// Spec: KWF-T4X9P FRK-WASM-022 Scope: Unit
func TestFRK_WASM_022_RenderHTML_EscapesTextAndAttributes(t *testing.T) {
	tree := El("a", map[string]string{"href": `/?x=1&x="2"`}, Text(`<b> & friends`))
	out := RenderHTML(tree)

	if strings.Count(out, "&lt;b&gt; &amp; friends") != 1 {
		t.Fatalf("text not escaped: %q", out)
	}
	if !strings.Contains(out, `href="/?x=1&amp;x=&#34;2&#34;"`) {
		t.Fatalf("attribute value not escaped: %q", out)
	}
}

// Spec: KWF-T4X9P FRK-WASM-022 Scope: Unit
func TestFRK_WASM_022_RenderHTML_VoidElementsAndDeterministicOrder(t *testing.T) {
	img := RenderHTML(El("img", map[string]string{"src": "a.png", "alt": "A"}))
	if img != `<img alt="A" src="a.png">` {
		t.Fatalf("void element output = %q", img)
	}

	nested := RenderHTML(El("ul", nil, El("li", nil, Text("one")), El("li", nil, Text("two"))))
	want := "<ul><li>one</li><li>two</li></ul>"
	if nested != want {
		t.Fatalf("nested output = %q, want %q", nested, want)
	}
}

// Spec: KWF-T4X9P FRK-WASM-022 Scope: Unit
func TestFRK_WASM_022_RenderHTML_ComponentPlaceholderCarriesNameAndProps(t *testing.T) {
	out := RenderHTML(Component("Counter", "c1", map[string]string{"start": "0"}))
	want := `<kiw-component name="Counter" data-kiw-key="c1" start="0"></kiw-component>`
	if out != want {
		t.Fatalf("component placeholder = %q, want %q", out, want)
	}

	if RenderHTML(nil) != "" {
		t.Fatalf("nil tree should render empty")
	}
}
