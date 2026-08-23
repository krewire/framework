package ssg

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/krewire/framework/ui"
)

func TestLayoutFromUIBuildsPage(t *testing.T) {
	s := New().Funcs(template.FuncMap{
		"html": func(v any) template.HTML {
			return template.HTML(fmt.Sprint(v))
		},
	})

	// A site shell composed entirely from reusable ui components.
	shell := ui.Layout{
		Description: "Built with the ui layout system.",
		Header: ui.Header{
			Brand:   template.HTML(`<a class="nav-brand" href="/">krewire</a>`),
			Nav:     ui.Nav{Links: []ui.NavLink{{Text: "Docs", URL: "/docs"}}}.Render(),
			Actions: ui.ThemeToggle{Theme: &ui.Theme{}}.Render(),
		}.Render(),
		Main: template.HTML(""),
		Footer: ui.Footer{
			Left:  template.HTML(`<span>krewire</span>`),
			Right: template.HTML("<p>© 2026</p>"),
		}.Render(),
		Theme: &ui.Theme{},
	}

	layout, err := LayoutFromUI("main", shell)
	if err != nil {
		t.Fatal(err)
	}
	s.Layout(layout)
	s.Component(Component{Name: "home", Body: `<section data-kiw-component="home"><h1>Home</h1></section>`})
	s.Page(Page{Path: "/", Title: "Home", Layout: "main", Root: "home"})

	out := t.TempDir()
	if _, err := s.Build(out); err != nil {
		t.Fatal(err)
	}

	page, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(page)
	for _, want := range []string{
		"<!DOCTYPE html>",
		"<title>Home</title>",
		`content="Built with the ui layout system."`,
		`<a class="nav-brand" href="/">krewire</a>`,
		`<a class="nav-link" href="/docs">Docs</a>`,
		"data-theme-toggle",
		`<section data-kiw-component="home">`,
		"<h1>Home</h1>",
		`data-kiw-layout="main"`,
		`data-ui-component="header"`,
		`data-ui-component="footer"`,
		"krewireTheme",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("index.html missing %q", want)
		}
	}
}

func TestLayoutFromUIRejectsBadTemplate(t *testing.T) {
	l := ui.Layout{Head: template.HTML(`{{if}}`)}
	if _, err := LayoutFromUI("bad", l); err == nil {
		t.Error("expected parse error for invalid template content")
	}
}
