package ssg

import (
	"strings"
	"testing"
)

func TestScopeCSS(t *testing.T) {
	in := `
h1 { color: red; }
h1, .x a:hover { margin: 0; }
:root { --a: 1; }
@media (max-width: 48rem) { .m { padding: 0; } }
@keyframes spin { from { transform: rotate(0); } to { transform: rotate(360deg); } }
@font-face { font-family: X; src: url(x.woff2); }
.x::before { content: "a, b"; }
`
	got := scopeCSS(`[data-kiw-component="c"]`, in, true)
	for _, want := range []string{
		`[data-kiw-component="c"] h1, [data-kiw-component="c"]h1{ color: red; }`,
		`[data-kiw-component="c"] h1, [data-kiw-component="c"]h1, [data-kiw-component="c"] .x a:hover{ margin: 0; }`,
		`:root{ --a: 1; }`,
		`@media (max-width: 48rem) {[data-kiw-component="c"] .m, [data-kiw-component="c"].m{ padding: 0; }}`,
		`@keyframes spin { from { transform: rotate(0); } to { transform: rotate(360deg); } }`,
		`@font-face { font-family: X; src: url(x.woff2); }`,
		`[data-kiw-component="c"] .x::before, [data-kiw-component="c"].x::before{ content: "a, b"; }`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("scoped CSS missing %q", want)
		}
	}
	if strings.Contains(got, `[data-kiw-component="c"] :root`) {
		t.Error(":root must not be scoped")
	}
	layout := scopeCSS(`[data-kiw-layout="l"]`, `body { margin: 0; }`, false)
	if layout != `[data-kiw-layout="l"] body{ margin: 0; }` {
		t.Errorf("layout scoping must not compound the root: %q", layout)
	}
}

func TestScopeFragmentInjectsRoot(t *testing.T) {
	got, err := scopeFragment("c", `<p>hi</p><p>bye</p>`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `<p data-kiw-component="c">hi</p>`) {
		t.Errorf("root element not scoped: %s", got)
	}
	if strings.Contains(string(got), `<p data-kiw-component="c">bye</p>`) {
		t.Error("non-root elements must not carry the scope attribute")
	}
}

func TestScopeDocumentInjectsHtml(t *testing.T) {
	got, err := scopeDocument("base", "<!DOCTYPE html><html><head></head><body>x</body></html>")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `<!DOCTYPE html>`) {
		t.Error("doctype lost")
	}
	if !strings.Contains(string(got), `<html data-kiw-layout="base">`) {
		t.Errorf("html element not scoped: %s", got)
	}
}
