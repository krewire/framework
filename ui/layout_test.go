package ui

import (
	"html/template"
	"strings"
	"testing"

	ftest "github.com/krewire/framework/test"
)

func TestLayoutRenderCompleteDocument(t *testing.T) {
	l := Layout{
		Title:       "Krewire",
		Description: "A meta-framework.",
		Header:      template.HTML(`<header data-ui-component="header"><span>brand</span></header>`),
		Main:        template.HTML("<p>content</p>"),
		Footer:      template.HTML(`<footer data-ui-component="footer"><span>footer</span></footer>`),
		Theme:       &Theme{},
	}
	out := string(l.Render())
	for _, want := range []string{
		"<!DOCTYPE html>",
		`<html lang="en"`,
		"<head>",
		"<title>Krewire</title>",
		`content="A meta-framework."`,
		"krewireTheme",
		"</head>",
		"<body",
		`<header data-ui-component="header"><span>brand</span></header>`,
		`<main data-ui-component="main">`,
		`<footer data-ui-component="footer"><span>footer</span></footer>`,
		"</body>",
		"</html>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("layout missing %q in output", want)
		}
	}
}

func TestLayoutOmittedRegions(t *testing.T) {
	l := Layout{Main: template.HTML("<p>content</p>")}
	out := string(l.Render())
	if strings.Contains(out, "<header") || strings.Contains(out, "<footer") || strings.Contains(out, "<aside") {
		t.Errorf("omitted regions must not render, got: %q", out)
	}
}

func TestLayoutWithHeadAndAttrs(t *testing.T) {
	l := Layout{Title: "T"}
	l = l.WithHead(template.HTML(`<link rel="stylesheet" href="/x.css">`))
	l = l.WithBodyAttrs(template.HTML(`data-page="home"`))
	out := string(l.Render())
	if !strings.Contains(out, `<link rel="stylesheet" href="/x.css">`) {
		t.Error("custom head not injected")
	}
	if !strings.Contains(out, `data-page="home"`) {
		t.Error("custom body attrs not injected")
	}
}

func TestLayoutEscapesTitle(t *testing.T) {
	l := Layout{Title: `<script>alert(1)</script>`}
	out := string(l.Render())
	if strings.Contains(out, `<script>alert(1)</script>`) {
		t.Error("title must be escaped")
	}
}

func TestLayoutFromConfig(t *testing.T) {
	cfg := &LayoutConfig{
		Title:       "Config Site",
		Description: "desc",
		Head:        template.HTML(`<meta name="generator" content="krewire">`),
		Stylesheets: []string{"/assets/ui.css"},
		Header: &HeaderConfig{
			Brand:       template.HTML(`<a href="/">krewire</a>`),
			Nav:         &NavConfig{Links: []NavLinkConfig{{Text: "Docs", URL: "/docs"}}},
			ThemeToggle: true,
		},
		Footer: &FooterConfig{Right: template.HTML("<p>© 2026</p>")},
		Main:   ComponentRef{HTML: template.HTML("<p>from config</p>")},
		Theme:  &ThemeConfig{Default: "dark", Light: map[string]string{"primary": "#111111"}},
	}
	l := LayoutFromConfig(cfg)
	if l.Title != "Config Site" {
		t.Errorf("title = %q", l.Title)
	}
	if len(l.Stylesheets) != 1 || l.Stylesheets[0] != "/assets/ui.css" {
		t.Errorf("stylesheets = %v", l.Stylesheets)
	}
	if string(l.Main) != "<p>from config</p>" {
		t.Errorf("main = %q", l.Main)
	}
	if string(l.Header) == "" {
		t.Error("header must render from config")
	}
	if !strings.Contains(string(l.Header), `class="ui-header"`) {
		t.Error("header must render the ui header shell")
	}
	if !strings.Contains(string(l.Header), `data-theme-toggle`) {
		t.Error("theme_toggle must inject the theme button")
	}
	if !strings.Contains(string(l.Footer), `<p>© 2026</p>`) {
		t.Error("footer must render right slot")
	}
	if l.Theme == nil || l.Theme.Default != "dark" {
		t.Fatalf("theme not attached: %+v", l.Theme)
	}
	if string(l.Theme.Light.Primary) != "#111111" {
		t.Errorf("light primary = %q", l.Theme.Light.Primary)
	}
	if string(l.Theme.Light.Base1) != string(DefaultLightPalette.Base1) {
		t.Error("unset tokens must fall back to defaults")
	}
}

func TestLayoutRendersStylesheets(t *testing.T) {
	l := Layout{Title: "T", Stylesheets: []string{"/assets/ui.css", "/assets/style.css"}}
	out := string(l.Render())
	if !strings.Contains(out, `<link rel="stylesheet" href="/assets/ui.css">`) {
		t.Error("stylesheets must emit link tags")
	}
	if !strings.Contains(out, `<link rel="stylesheet" href="/assets/style.css">`) {
		t.Error("second stylesheet must emit too")
	}
}

// --- UI Testing Framework (KWF-TEST-U9K3M) — Layout with HTMLAssert ---

// Spec: KWF-TEST-U9K3M KWF-TST-U9K-010 Scope: Unit
func TestKWF_TST_U9K_010_Layout_HTMLAssert(t *testing.T) {
	ftest.Spec(t, "KWF-TEST-U9K3M", "KWF-TST-U9K-010")
	l := Layout{
		Title:  "Krewire",
		Header: template.HTML(`<header data-ui-component="header"><span>brand</span></header>`),
		Main:   template.HTML(`<p>content</p><nav><a href="/a">a</a><a href="/b">b</a></nav>`),
		Aside:  template.HTML(`<nav>side</nav>`),
		Footer: template.HTML(`<footer data-ui-component="footer"><span>footer</span></footer>`),
	}
	html := string(l.Render())
	ftest.HTML(t, html).
		Has("html", 1).
		Has("head", 1).
		Has("title", 1).
		HasText("title", "Krewire").
		Has("body", 1).
		Has("header", 1).
		Has("main", 1).
		Has("aside", 1).
		Has("footer", 1).
		Has("nav a", 2).
		Attr("html", "lang", "en")
}

// Spec: KWF-TEST-U9K3M KWF-TST-U9K-011 Scope: Unit
func TestKWF_TST_U9K_011_Layout_Snapshot(t *testing.T) {
	ftest.Spec(t, "KWF-TEST-U9K3M", "KWF-TST-U9K-011")
	l := Layout{
		Title: "Snapshot",
		Main:  template.HTML("<p>hello</p>"),
	}
	ftest.Snapshot(t, "ui_layout_snapshot", string(l.Render()))
}

// Spec: KWF-TEST-U9K3M KWF-TST-U9K-011 Scope: Unit
func TestKWF_TST_U9K_011_Layout_OmittedRegions_HTMLAssert(t *testing.T) {
	ftest.Spec(t, "KWF-TEST-U9K3M", "KWF-TST-U9K-011")
	l := Layout{Main: template.HTML("<p>content</p>")}
	html := string(l.Render())
	a := ftest.HTML(t, html)
	if a.Count("header") != 0 {
		t.Errorf("header should be omitted, got %d", a.Count("header"))
	}
	if a.Count("footer") != 0 {
		t.Errorf("footer should be omitted")
	}
	if a.Count("aside") != 0 {
		t.Errorf("aside should be omitted")
	}
	a.Has("main", 1).Has("p", 1).HasText("p", "content")
}

// Spec: KWF-TEST-U9K3M KWF-TST-U9K-012 Scope: Unit
func TestKWF_TST_U9K_012_Layout_ThemeIntegration(t *testing.T) {
	ftest.Spec(t, "KWF-TEST-U9K3M", "KWF-TST-U9K-012")
	th := &Theme{Default: "dark", Light: Palette{Primary: "#111111"}}
	l := Layout{Title: "T", Main: template.HTML("<p>x</p>"), Theme: th}
	html := string(l.Render())
	ftest.HTML(t, html).Has("html", 1).Attr("html", "data-theme", "auto")
	ftest.ThemeSnapshot(t, "ui_layout_theme", html)
}

// Spec: KWF-TEST-U9K3M KWF-TST-U9K-013 Scope: Unit
func TestKWF_TST_U9K_013_Layout_GoldenHTML(t *testing.T) {
	ftest.Spec(t, "KWF-TEST-U9K3M", "KWF-TST-U9K-013")
	l := Layout{
		Title:       "Golden",
		Description: "desc",
		Stylesheets: []string{"/assets/app.abc123.css", "/assets/ui.css"},
		Main:        template.HTML("<p>golden</p>"),
	}
	html := string(l.Render())
	ftest.GoldenHTML(t, "ui_layout_golden", html)
}
