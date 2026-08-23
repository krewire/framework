package ui

import (
	"html/template"
	"strings"
	"testing"
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
