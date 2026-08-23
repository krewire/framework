package ui

import (
	"html/template"
	"strings"
	"testing"
)

func TestHeaderRender(t *testing.T) {
	h := Header{
		Brand:   template.HTML(`<a href="/">krewire</a>`),
		Nav:     Nav{Links: []NavLink{{Text: "Docs", URL: "/docs"}, {Text: "GitHub", URL: "https://github.com/krewire", Target: "_blank"}}}.Render(),
		Actions: ThemeToggle{Theme: &Theme{}}.Render(),
	}
	out := string(h.Render())
	for _, want := range []string{
		`<header class="ui-header"`,
		`header-inner`,
		`header-brand`,
		`<a href="/">krewire</a>`,
		`header-nav`,
		`<nav class="nav"`,
		`<a class="nav-link" href="/docs">Docs</a>`,
		`header-actions`,
		`data-theme-toggle`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("header missing %q", want)
		}
	}
}

func TestNavActiveState(t *testing.T) {
	n := Nav{Links: []NavLink{{Text: "Home", URL: "/", Active: true}, {Text: "Docs", URL: "/docs"}}}
	out := string(n.Render())
	if !strings.Contains(out, `is-active`) {
		t.Error("active link must carry is-active class")
	}
	if !strings.Contains(out, `aria-current="page"`) {
		t.Error("active link must carry aria-current")
	}
}

func TestNavEscapesInput(t *testing.T) {
	n := Nav{Links: []NavLink{{Text: `<img src=x onerror=alert(1)>`, URL: `"><script>`}}}
	out := string(n.Render())
	if strings.Contains(out, `<img src=x onerror=alert(1)>`) || strings.Contains(out, `<script>`) {
		t.Error("nav must escape text and URL")
	}
}

func TestFooterRender(t *testing.T) {
	f := Footer{
		Left:  template.HTML(`<span>krewire</span>`),
		Right: template.HTML(`<p>© 2026</p>`),
	}
	out := string(f.Render())
	for _, want := range []string{
		`<footer class="ui-footer"`,
		`footer-left`,
		`footer-right`,
		`<span>krewire</span>`,
		`<p>© 2026</p>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("footer missing %q", want)
		}
	}
	if strings.Contains(out, "footer-center") {
		t.Error("empty center slot must not render")
	}
}

func TestContainerRender(t *testing.T) {
	out := string(Container{Content: template.HTML("<p>hi</p>")}.Render())
	if !strings.Contains(out, `<div class="container" data-ui-component="container">`) || !strings.Contains(out, "<p>hi</p>") {
		t.Errorf("container output wrong: %q", out)
	}
}

func TestGridRenderColumns(t *testing.T) {
	out := string(Grid{Columns: 3, Content: template.HTML("<div>a</div>")}.Render())
	if !strings.Contains(out, `class="grid grid-3"`) {
		t.Errorf("grid class wrong: %q", out)
	}
	if !strings.Contains(out, `<div>a</div>`) {
		t.Error("grid content missing")
	}
	// invalid column counts fall back to 4
	if out := string(Grid{Columns: 99}.Render()); !strings.Contains(out, "grid-4") {
		t.Error("invalid columns must default to 4")
	}
}

func TestCardRender(t *testing.T) {
	c := Card{
		Header: template.HTML("<h3>Title</h3>"),
		Body:   template.HTML("<p>body</p>"),
		Footer: template.HTML("<a href=\"/\">more</a>"),
	}
	out := string(c.Render())
	for _, want := range []string{`<article class="card"`, "card-header", "card-body", "card-footer", "<h3>Title</h3>", "<p>body</p>"} {
		if !strings.Contains(out, want) {
			t.Errorf("card missing %q", want)
		}
	}
}

func TestHeroRender(t *testing.T) {
	h := Hero{
		Badge:    template.HTML("Go Native"),
		Title:    template.HTML("Build Static Sites at Go Speed."),
		Subtitle: template.HTML("Meta-framework."),
		CTALeft:  template.HTML(`<a href="/docs">Docs</a>`),
		Visual:   template.HTML(`<div class="terminal"></div>`),
	}
	out := string(h.Render())
	for _, want := range []string{
		`<section class="hero" data-ui-component="hero">`,
		"hero-badge",
		`<h1 class="hero-title">Build Static Sites at Go Speed.</h1>`,
		"hero-subtitle",
		"hero-cta",
		`<a href="/docs">Docs</a>`,
		"hero-visual",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("hero missing %q", want)
		}
	}
}

func TestBadgeRender(t *testing.T) {
	if out := string(Badge{Text: "Go Native", Variant: "primary"}.Render()); !strings.Contains(out, `class="badge badge-primary"`) {
		t.Errorf("badge wrong: %q", out)
	}
	if out := string(Badge{Text: "warn"}.Render()); !strings.Contains(out, "badge-primary") {
		t.Errorf("default variant must be primary: %q", out)
	}
	if out := string(Badge{Text: "<b>"}.Render()); strings.Contains(out, "<b>") {
		t.Error("badge text must be escaped")
	}
}

func TestButtonRender(t *testing.T) {
	link := Button{Text: "Docs", URL: "/docs", Variant: "ghost", Target: "_blank", Rel: "noopener"}.Render()
	if !strings.Contains(string(link), `<a class="btn btn-ghost" data-ui-component="button" href="/docs" target="_blank" rel="noopener">Docs</a>`) {
		t.Errorf("link button wrong: %q", link)
	}
	btn := Button{Text: "Save", Variant: "primary"}.Render()
	if !strings.Contains(string(btn), `<button type="button" class="btn btn-primary" data-ui-component="button">Save</button>`) {
		t.Errorf("button wrong: %q", btn)
	}
}

func TestSlotConcatenates(t *testing.T) {
	out := Slot(template.HTML("<a>1</a>"), template.HTML("<b>2</b>"))
	if string(out) != "<a>1</a><b>2</b>" {
		t.Errorf("slot = %q", out)
	}
}

func TestComponentsCSSIncludesAll(t *testing.T) {
	css := ComponentsCSS()
	for _, sel := range []string{
		".container",
		".grid-1",
		".ui-header",
		".nav-link",
		".ui-footer",
		".card",
		".section",
		".hero",
		".badge-primary",
		".btn-primary",
		".theme-toggle-wrap",
	} {
		if !strings.Contains(css, sel) {
			t.Errorf("ComponentsCSS missing %q", sel)
		}
	}
}
