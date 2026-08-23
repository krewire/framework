package ssg

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/krewire/framework/ui"
)

func testConfig() *Config {
	return &Config{
		Title:       "Krewire",
		Description: "A meta-framework written in Go.",
		Theme: &ThemeConfig{
			Default: "auto",
			Light:   map[string]string{"primary": "#00c853"},
			Dark:    map[string]string{"primary": "#5cff8e"},
		},
		Components: []ComponentConfig{
			{Name: "badge", Body: `<span class="badge">{{html .}}</span>`, Style: `.badge { color: var(--primary); }`},
			{Name: "hero", Body: `<section class="hero">{{component "badge" .Feature}}<h1>{{.Title}}</h1></section>`, Style: `h1 { font-size: 2rem; }`},
		},
		Layouts: []LayoutConfig{
			{Name: "base", Body: `<!DOCTYPE html><html><head><title>{{.Title}}</title><link rel="stylesheet" href="/assets/style.css">{{if .Data.Theme}}{{.Data.Theme.Script}}{{end}}</head><body>{{.Content}}</body></html>`},
		},
		Pages: []PageConfig{
			{Path: "/", Title: "Home", Layout: "base", Root: "hero"},
		},
		Data: map[string]any{
			"Feature": "<strong>fast</strong>",
		},
	}
}

func TestConfigSiteBuilds(t *testing.T) {
	out := t.TempDir()
	created, err := BuildFromConfig(testConfig(), out)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 3 {
		t.Fatalf("created = %v, want index.html, style.css, theme.css", created)
	}
	page, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`data-kiw-component="hero"`,
		`data-kiw-component="badge"`,
		`data-kiw-layout="base"`,
		`<h1>Krewire</h1>`,
		`<strong>fast</strong>`, // trusted HTML via the html helper
		`window.krewireTheme`,
		"<!DOCTYPE html>",
	} {
		if !strings.Contains(string(page), want) {
			t.Errorf("index.html missing %q", want)
		}
	}
	for _, asset := range []string{"assets/style.css", "assets/theme.css"} {
		if _, err := os.Stat(filepath.Join(out, filepath.FromSlash(asset))); err != nil {
			t.Errorf("missing asset %s: %v", asset, err)
		}
	}
}

func TestConfigSiteEscapesUntrustedData(t *testing.T) {
	cfg := testConfig()
	cfg.Components = []ComponentConfig{
		{Name: "badge", Body: `<span class="badge">{{.}}</span>`},
		{Name: "hero", Body: `<section class="hero">{{component "badge" .Feature}}<h1>{{.Title}}</h1></section>`},
	}
	cfg.Data = map[string]any{"Feature": "<script>alert(1)</script>"}
	out := t.TempDir()
	if _, err := BuildFromConfig(cfg, out); err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(page), "<script>alert(1)</script>") {
		t.Error("plain data must be escaped, not trusted")
	}
}

func TestConfigThemePaletteOverrides(t *testing.T) {
	cfg := testConfig()
	th := cfg.theme()
	if string(th.Light.Primary) != "#00c853" {
		t.Errorf("light primary = %q, want #00c853", th.Light.Primary)
	}
	if string(th.Dark.Primary) != "#5cff8e" {
		t.Errorf("dark primary = %q, want #5cff8e", th.Dark.Primary)
	}
	if th.Light.Base1 != "" {
		t.Error("unset tokens must stay empty and fall back at render time")
	}
	if !strings.Contains(string(th.Style()), "--base-1: "+string(ui.DefaultLightPalette.Base1)) {
		t.Error("rendered theme style must fill unset tokens from the ui defaults")
	}
}

func TestConfigNoThemeOmitsThemeAsset(t *testing.T) {
	cfg := testConfig()
	cfg.Theme = nil
	out := t.TempDir()
	if _, err := BuildFromConfig(cfg, out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "assets", "theme.css")); !os.IsNotExist(err) {
		t.Error("theme.css must be omitted when Theme is unset")
	}
}

func TestConfigPageDataMergesOverrides(t *testing.T) {
	cfg := testConfig()
	cfg.Pages = []PageConfig{
		{Path: "/", Title: "Home", Layout: "base", Root: "hero", Data: map[string]any{"Feature": "<b>page</b>"}},
	}
	out := t.TempDir()
	if _, err := BuildFromConfig(cfg, out); err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), "<b>page</b>") {
		t.Error("page-specific data must override site-wide data")
	}
}

func TestConfigUIShellLayout(t *testing.T) {
	cfg := &Config{
		Title:       "Krewire",
		Description: "Built from a ui shell.",
		Theme:       &ThemeConfig{Default: "auto"},
		Layouts: []LayoutConfig{
			{
				Name: "main",
				UI: &ui.LayoutConfig{
					Description: "Built from a ui shell.",
					Stylesheets: []string{"/assets/ui.css"},
					Header: &ui.HeaderConfig{
						Brand:       template.HTML(`<a href="/">krewire</a>`),
						Nav:         &ui.NavConfig{Links: []ui.NavLinkConfig{{Text: "Docs", URL: "/docs"}}},
						ThemeToggle: true,
					},
					Footer: &ui.FooterConfig{Right: template.HTML("<p>© 2026</p>")},
				},
			},
		},
		Components: []ComponentConfig{
			{Name: "home", Body: `<section class="home"><h1>{{.Title}}</h1></section>`},
		},
		Pages: []PageConfig{
			{Path: "/", Title: "Home", Layout: "main", Root: "home"},
		},
	}
	out := t.TempDir()
	created, err := BuildFromConfig(cfg, out)
	if err != nil {
		t.Fatal(err)
	}
	var hasUIcss bool
	for _, p := range created {
		if p == "assets/ui.css" {
			hasUIcss = true
		}
	}
	if !hasUIcss {
		t.Fatalf("ui shell sites must emit assets/ui.css, got %v", created)
	}
	page, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(page)
	for _, want := range []string{
		`<title>Home</title>`,
		`<a href="/">krewire</a>`,
		`<a class="nav-link" href="/docs">Docs</a>`,
		`data-theme-toggle`,
		`<p>© 2026</p>`,
		`data-kiw-layout="main"`,
		`<h1>Krewire</h1>`,
		`data-ui-component="header"`,
		`data-ui-component="footer"`,
		`href="/assets/ui.css"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("index.html missing %q", want)
		}
	}
}

func TestConfigUIShellInvalidFailsBuild(t *testing.T) {
	cfg := &Config{
		Layouts: []LayoutConfig{
			{Name: "main", UI: &ui.LayoutConfig{Head: template.HTML(`{{if}}`)}},
		},
		Pages: []PageConfig{
			{Path: "/", Title: "Home", Layout: "main", Root: "home"},
		},
	}
	if _, err := BuildFromConfig(cfg, t.TempDir()); err == nil {
		t.Error("invalid ui shell must fail the build")
	}
}

func TestConfigPageRootResolvesViaRegistry(t *testing.T) {
	cfg := &Config{
		Title: "Krewire",
		Layouts: []LayoutConfig{
			{Name: "base", Body: `<!DOCTYPE html><html><body>{{.Content}}</body></html>`},
		},
		Pages: []PageConfig{
			{Path: "/", Title: "Home", Layout: "base", Root: "Hero", Data: map[string]any{"title": "Go Native", "subtitle": "Meta-framework."}},
		},
	}
	out := t.TempDir()
	if _, err := BuildFromConfig(cfg, out); err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<section class="hero" data-ui-component="hero">`,
		`<h1 class="hero-title">Go Native</h1>`,
	} {
		if !strings.Contains(string(page), want) {
			t.Errorf("index.html missing %q", want)
		}
	}
}

func TestConfigPageRootUnknownRegistryComponentFails(t *testing.T) {
	cfg := &Config{
		Layouts: []LayoutConfig{
			{Name: "base", Body: `<!DOCTYPE html><html><body>{{.Content}}</body></html>`},
		},
		Pages: []PageConfig{
			{Path: "/", Title: "Home", Layout: "base", Root: "DoesNotExist"},
		},
	}
	if _, err := BuildFromConfig(cfg, t.TempDir()); err == nil {
		t.Error("undefined root component must fail the build")
	}
}
