package ui

import (
	"html/template"
	"strings"
	"testing"
)

func TestDefaultRegistryHasStandardComponents(t *testing.T) {
	r := Default()
	for _, name := range []string{
		"Header", "Footer", "Nav", "Container", "Grid", "Card", "Section",
		"Hero", "ThemeToggle", "Badge", "Button",
	} {
		if _, ok := r.Get(name); !ok {
			t.Errorf("default registry missing %q", name)
		}
	}
}

func TestRegistryRenderHeroMatchesBuiltin(t *testing.T) {
	r := Default()
	props := map[string]any{
		"title":    "Go Static",
		"subtitle": "Meta-framework.",
	}
	want := Hero{Title: template.HTML("Go Static"), Subtitle: template.HTML("Meta-framework.")}.Render()
	got, err := r.Render("Hero", props)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("registry Hero = %q, want %q", got, want)
	}
}

func TestRegistryRenderTypedProps(t *testing.T) {
	r := Default()
	h := Hero{Title: template.HTML("Typed"), Badge: template.HTML("v1")}
	got, err := r.Render("Hero", h)
	if err != nil {
		t.Fatal(err)
	}
	if got != h.Render() {
		t.Errorf("typed props Hero = %q, want %q", got, h.Render())
	}
}

func TestRegistryUnknownComponent(t *testing.T) {
	r := Default()
	if _, err := r.Render("Nope", nil); err == nil {
		t.Fatal("expected error for unknown component")
	}
	if _, ok := r.Get("Nope"); ok {
		t.Fatal("Get must return false for unknown component")
	}
}

func TestRegistryRejectsUnknownProps(t *testing.T) {
	r := Default()
	_, err := r.Render("Hero", map[string]any{"title": "x", "nope": "y"})
	if err == nil {
		t.Fatal("expected error for unknown prop key")
	}
}

func TestRegistryOverrideByName(t *testing.T) {
	r := Default()
	r.Register("Hero", typed(func(b Badge) template.HTML { return b.Render() }))
	got, err := r.Render("Hero", map[string]any{"text": "custom", "variant": "success"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `class="badge badge-success"`) {
		t.Errorf("override not applied: %q", got)
	}
}

func TestRegistryRenderCarriesScopeAttr(t *testing.T) {
	r := Default()
	for _, tc := range []struct {
		name  string
		props any
	}{
		{"Hero", map[string]any{"title": "Hi"}},
		{"Grid", map[string]any{"columns": 3}},
		{"Button", map[string]any{"text": "Go", "url": "/docs"}},
	} {
		out, err := r.Render(tc.name, tc.props)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if !strings.Contains(string(out), `data-ui-component="`+strings.ToLower(tc.name)+`"`) {
			t.Errorf("%s output missing data-ui-component scope: %q", tc.name, out)
		}
	}
}

func TestRegistryLayoutComponentEquivalence(t *testing.T) {
	r := Default()
	out, err := r.Render("Footer", map[string]any{"right": "<p>© 2026</p>"})
	if err != nil {
		t.Fatal(err)
	}
	want := Footer{Right: template.HTML("<p>© 2026</p>")}.Render()
	if out != want {
		t.Errorf("registry Footer = %q, want %q", out, want)
	}
}
