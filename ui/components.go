package ui

import (
	"fmt"
	"html/template"
	"strings"
)

// Slot concatenates multiple pre-rendered HTML fragments into one value.
func Slot(parts ...template.HTML) template.HTML {
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(string(p))
	}
	return template.HTML(b.String())
}

// NavLink is a single navigation entry.
type NavLink struct {
	Text   string `json:"text"`
	URL    string `json:"url"`
	Active bool   `json:"active"`
	Target string `json:"target"` // optional: "_blank"
	Rel    string `json:"rel"`    // optional: "noopener"
	Class  string `json:"class"`  // optional: extra CSS classes on the anchor
}

// Nav renders a navigation bar with active-state support.
type Nav struct {
	Links []NavLink `json:"links"`
	Label string    `json:"label"` // aria-label for the <nav> element
}

// Render produces <nav><ul><li><a>...</a></li></ul></nav>.
func (n Nav) Render() template.HTML {
	label := n.Label
	if label == "" {
		label = "Main navigation"
	}
	var b strings.Builder
	b.WriteString(`<nav class="nav" data-ui-component="nav" aria-label="` + template.HTMLEscapeString(label) + `">`)
	b.WriteString(`<ul class="nav-list">`)
	for _, l := range n.Links {
		b.WriteString(`<li class="nav-item`)
		if l.Active {
			b.WriteString(" is-active")
		}
		b.WriteString(`">`)
		b.WriteString(`<a class="nav-link`)
		if l.Class != "" {
			b.WriteString(` ` + template.HTMLEscapeString(l.Class))
		}
		b.WriteString(`" href="` + template.HTMLEscapeString(l.URL) + `"`)
		if l.Active {
			b.WriteString(` aria-current="page"`)
		}
		if l.Target != "" {
			b.WriteString(` target="` + template.HTMLEscapeString(l.Target) + `"`)
		}
		if l.Rel != "" {
			b.WriteString(` rel="` + template.HTMLEscapeString(l.Rel) + `"`)
		}
		b.WriteString(`>` + template.HTMLEscapeString(l.Text) + `</a>`)
		b.WriteString(`</li>`)
	}
	b.WriteString(`</ul>`)
	b.WriteString(`</nav>`)
	return template.HTML(b.String())
}

// Header is a top bar with Brand, Nav, and Actions slots.
type Header struct {
	Brand   template.HTML `json:"brand"`
	Nav     template.HTML `json:"nav"`
	Actions template.HTML `json:"actions"`
}

// Render produces <header><div class="header-inner">…</div></header>.
func (h Header) Render() template.HTML {
	var b strings.Builder
	b.WriteString(`<header class="ui-header" data-ui-component="header">`)
	b.WriteString(`<div class="header-inner">`)
	if h.Brand != "" {
		b.WriteString(`<div class="header-brand">`)
		b.WriteString(string(h.Brand))
		b.WriteString(`</div>`)
	}
	if h.Nav != "" {
		b.WriteString(`<div class="header-nav">`)
		b.WriteString(string(h.Nav))
		b.WriteString(`</div>`)
	}
	if h.Actions != "" {
		b.WriteString(`<div class="header-actions">`)
		b.WriteString(string(h.Actions))
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
	b.WriteString(`</header>`)
	return template.HTML(b.String())
}

// Footer is a bottom bar with Left, Center, and Right slots.
type Footer struct {
	Left   template.HTML `json:"left"`
	Center template.HTML `json:"center"`
	Right  template.HTML `json:"right"`
}

// Render produces <footer><div class="footer-inner">…</div></footer>.
func (f Footer) Render() template.HTML {
	var b strings.Builder
	b.WriteString(`<footer class="ui-footer" data-ui-component="footer">`)
	b.WriteString(`<div class="footer-inner">`)
	if f.Left != "" {
		b.WriteString(`<div class="footer-left">`)
		b.WriteString(string(f.Left))
		b.WriteString(`</div>`)
	}
	if f.Center != "" {
		b.WriteString(`<div class="footer-center">`)
		b.WriteString(string(f.Center))
		b.WriteString(`</div>`)
	}
	if f.Right != "" {
		b.WriteString(`<div class="footer-right">`)
		b.WriteString(string(f.Right))
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
	b.WriteString(`</footer>`)
	return template.HTML(b.String())
}

// Container is a max-width wrapper with responsive padding.
type Container struct {
	Content template.HTML `json:"content"`
}

// Render produces <div class="container">…</div>.
func (c Container) Render() template.HTML {
	return template.HTML(`<div class="container" data-ui-component="container">` + string(c.Content) + `</div>`)
}

// Grid is a responsive CSS grid.
type Grid struct {
	Columns int           `json:"columns"` // 1..6, defaults to 4 when <= 0
	Gap     string        `json:"gap"`     // optional CSS gap, e.g. "1.5rem"
	Content template.HTML `json:"content"`
}

// Render produces <div class="grid grid-{n}">…</div>.
func (g Grid) Render() template.HTML {
	cols := g.Columns
	if cols <= 0 || cols > 6 {
		cols = 4
	}
	style := ""
	if g.Gap != "" {
		style = ` style="gap:` + template.HTMLEscapeString(g.Gap) + `"`
	}
	return template.HTML(fmt.Sprintf(`<div class="grid grid-%d" data-ui-component="grid"%s>`, cols, style) + string(g.Content) + `</div>`)
}

// Card is a bordered container with optional header, body, and footer.
type Card struct {
	Header template.HTML `json:"header"`
	Body   template.HTML `json:"body"`
	Footer template.HTML `json:"footer"`
}

// Render produces <article class="card">…</article>.
func (c Card) Render() template.HTML {
	var b strings.Builder
	b.WriteString(`<article class="card" data-ui-component="card">`)
	if c.Header != "" {
		b.WriteString(`<div class="card-header">`)
		b.WriteString(string(c.Header))
		b.WriteString(`</div>`)
	}
	if c.Body != "" {
		b.WriteString(`<div class="card-body">`)
		b.WriteString(string(c.Body))
		b.WriteString(`</div>`)
	}
	if c.Footer != "" {
		b.WriteString(`<div class="card-footer">`)
		b.WriteString(string(c.Footer))
		b.WriteString(`</div>`)
	}
	b.WriteString(`</article>`)
	return template.HTML(b.String())
}

// Section is a vertical-rhythm container.
type Section struct {
	Background string        `json:"background"` // optional CSS background value
	Content    template.HTML `json:"content"`
}

// Render produces <section>…</section>.
func (s Section) Render() template.HTML {
	if s.Background != "" {
		return template.HTML(`<section class="section" data-ui-component="section" style="background:` +
			template.HTMLEscapeString(s.Background) + `">` + string(s.Content) + `</section>`)
	}
	return template.HTML(`<section class="section" data-ui-component="section">` + string(s.Content) + `</section>`)
}

// Hero is a prominent top section with headline, subtext, and CTA slots.
type Hero struct {
	Badge    template.HTML `json:"badge"`
	Title    template.HTML `json:"title"`
	Subtitle template.HTML `json:"subtitle"`
	CTALeft  template.HTML `json:"cta_left"`
	CTARight template.HTML `json:"cta_right"`
	Visual   template.HTML `json:"visual"` // optional illustration/terminal
	Extra    template.HTML `json:"extra"`
}

// Render produces <section class="hero">…</section>.
func (h Hero) Render() template.HTML {
	var b strings.Builder
	b.WriteString(`<section class="hero" data-ui-component="hero">`)
	b.WriteString(`<div class="hero-inner">`)
	b.WriteString(`<div class="hero-text">`)
	if h.Badge != "" {
		b.WriteString(`<div class="hero-badge">`)
		b.WriteString(string(h.Badge))
		b.WriteString(`</div>`)
	}
	if h.Title != "" {
		b.WriteString(`<h1 class="hero-title">`)
		b.WriteString(string(h.Title))
		b.WriteString(`</h1>`)
	}
	if h.Subtitle != "" {
		b.WriteString(`<p class="hero-subtitle">`)
		b.WriteString(string(h.Subtitle))
		b.WriteString(`</p>`)
	}
	if h.CTALeft != "" || h.CTARight != "" {
		b.WriteString(`<div class="hero-cta">`)
		if h.CTALeft != "" {
			b.WriteString(string(h.CTALeft))
		}
		if h.CTARight != "" {
			b.WriteString(string(h.CTARight))
		}
		b.WriteString(`</div>`)
	}
	if h.Extra != "" {
		b.WriteString(string(h.Extra))
	}
	b.WriteString(`</div>`)
	if h.Visual != "" {
		b.WriteString(`<div class="hero-visual">`)
		b.WriteString(string(h.Visual))
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
	b.WriteString(`</section>`)
	return template.HTML(b.String())
}

// ThemeToggle wraps the theme button in a stable container.
type ThemeToggle struct {
	Theme *Theme `json:"theme"`
}

// Render returns the theme-switcher button.
func (t ThemeToggle) Render() template.HTML {
	if t.Theme == nil {
		t.Theme = &Theme{}
	}
	return template.HTML(`<div class="theme-toggle-wrap" data-ui-component="theme-toggle">` +
		string(t.Theme.Button()) + `</div>`)
}

// Badge is an inline label with semantic variants.
type Badge struct {
	Text    string `json:"text"`
	Variant string `json:"variant"` // primary, neutral, success, info, warning, error; default primary
}

// Render produces <span class="badge badge-{variant}">…</span>.
func (b Badge) Render() template.HTML {
	v := b.Variant
	if v == "" {
		v = "primary"
	}
	return template.HTML(`<span class="badge badge-` + template.HTMLEscapeString(v) +
		`" data-ui-component="badge">` + template.HTMLEscapeString(b.Text) + `</span>`)
}

// Button is a styled link or button.
type Button struct {
	Text    string `json:"text"`
	URL     string `json:"url"`     // when set, renders <a>
	Variant string `json:"variant"` // primary, ghost, outline; default primary
	Target  string `json:"target"`
	Rel     string `json:"rel"`
}

// Render produces an <a> or <button> element.
func (b Button) Render() template.HTML {
	cls := "btn btn-" + b.Variant
	if b.Variant == "" {
		cls = "btn btn-primary"
	}
	if b.URL != "" {
		attrs := ` href="` + template.HTMLEscapeString(b.URL) + `"`
		if b.Target != "" {
			attrs += ` target="` + template.HTMLEscapeString(b.Target) + `"`
		}
		if b.Rel != "" {
			attrs += ` rel="` + template.HTMLEscapeString(b.Rel) + `"`
		}
		return template.HTML(`<a class="` + cls + `" data-ui-component="button"` + attrs + `>` +
			template.HTMLEscapeString(b.Text) + `</a>`)
	}
	return template.HTML(`<button type="button" class="` + cls + `" data-ui-component="button">` +
		template.HTMLEscapeString(b.Text) + `</button>`)
}
