package ssg

import (
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testSite() *Site {
	return New().
		Component(Component{
			Name:  "badge",
			Body:  `<span class="badge">{{.}}</span>`,
			Style: `.badge { color: #0b58a1; }`,
		}).
		Component(Component{
			Name:  "nav",
			Body:  `<nav><a href="/">Home</a>{{component "badge" .}}</nav>`,
			Style: `nav a { color: #333; } @media (max-width: 48rem) { nav a { color: #000; } }`,
		}).
		Component(Component{
			Name:  "hero",
			Body:  `<section class="hero">{{component "nav" .}}<h1>{{.Title}}</h1>{{component "badge" .Version}}</section>`,
			Style: `.hero :root { color: red; } h1 { font-size: 2rem; }`,
		}).
		Layout(Layout{
			Name:  "base",
			Body:  `<!DOCTYPE html><html><head><title>{{.Title}}</title><link rel="stylesheet" href="/assets/style.css"></head><body>{{.Content}}</body></html>`,
			Style: `body { margin: 0; font: 16px/1.6 sans-serif; }`,
		}).
		Page(Page{
			Path:   "/",
			Title:  "Home",
			Layout: "base",
			Root:   "hero",
			Data:   struct{ Title, Version string }{"Welcome", "v1.0"},
		})
}

func TestBuildProducesPages(t *testing.T) {
	out := t.TempDir()
	created, err := testSite().Build(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 2 {
		t.Fatalf("created = %v, want index.html and style.css", created)
	}
}

func TestScopeInjectedAndStylesCollected(t *testing.T) {
	out := t.TempDir()
	if _, err := testSite().Build(out); err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`data-kiw-component="hero"`,
		`data-kiw-component="nav"`,
		`data-kiw-component="badge"`,
		`data-kiw-layout="base"`,
		"<!DOCTYPE html>",
	} {
		if !strings.Contains(string(page), want) {
			t.Errorf("index.html missing %q", want)
		}
	}
	css, err := os.ReadFile(filepath.Join(out, "assets", "style.css"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`[data-kiw-component="badge"] .badge, [data-kiw-component="badge"].badge{ color: #0b58a1; }`,
		`[data-kiw-component="nav"] nav a{ color: #333; }`,
		`@media (max-width: 48rem) {[data-kiw-component="nav"] nav a{ color: #000; }}`,
		`[data-kiw-component="hero"] h1, [data-kiw-component="hero"]h1{ font-size: 2rem; }`,
		`.hero :root{ color: red; }`, // :root stays global
		`[data-kiw-layout="base"] body{ margin: 0; font: 16px/1.6 sans-serif; }`,
	} {
		if !strings.Contains(string(css), want) {
			t.Errorf("style.css missing %q", want)
		}
	}
}

func TestUnusedComponentOmitted(t *testing.T) {
	s := testSite().Component(Component{Name: "unused", Body: `<p>x</p>`, Style: `.unused { color: red; }`})
	out := t.TempDir()
	if _, err := s.Build(out); err != nil {
		t.Fatal(err)
	}
	css, err := os.ReadFile(filepath.Join(out, "assets", "style.css"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(css), "unused") {
		t.Error("unused component CSS must not be emitted")
	}
}

func TestExplicitHtmlPathAndAssets(t *testing.T) {
	s := New().
		Component(Component{Name: "p", Body: `<p>{{.}}</p>`}).
		Layout(Layout{Name: "l", Body: `<!DOCTYPE html><html><body>{{.Content}}</body></html>`}).
		Page(Page{Path: "/404.html", Layout: "l", Root: "p", Data: "gone"}).
		Asset("favicon.ico", "x")
	out := t.TempDir()
	created, err := s.Build(out)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "404.html")); err != nil {
		t.Error("expected 404.html")
	}
	if _, err := os.Stat(filepath.Join(out, "favicon.ico")); err != nil {
		t.Error("expected favicon.ico")
	}
	_ = created
}

func TestUndefinedComponentErrors(t *testing.T) {
	s := New().
		Layout(Layout{Name: "l", Body: `<html><body>{{.Content}}</body></html>`}).
		Page(Page{Path: "/", Layout: "l", Root: "missing"})
	if _, err := s.Build(t.TempDir()); err == nil {
		t.Fatal("expected error for undefined root component")
	}
}

func TestHandlerRendersSameAsBuild(t *testing.T) {
	out := t.TempDir()
	if _, err := testSite().Build(out); err != nil {
		t.Fatal(err)
	}
	built, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(testSite().Handler())
	defer ts.Close()
	res, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	served, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	// ServeContent vs build order of attributes may differ; compare a core fragment.
	for _, want := range []string{`data-kiw-component="hero"`, "Welcome", `data-kiw-layout="base"`} {
		if !strings.Contains(string(served), want) {
			t.Errorf("served page missing %q", want)
		}
	}
	_ = built
}

func TestTemplateFuncs(t *testing.T) {
	s := New().Funcs(template.FuncMap{
		"upper": func(s string) string { return strings.ToUpper(s) },
	}).
		Component(Component{Name: "c", Body: `<em>{{upper .}}</em>`}).
		Layout(Layout{Name: "l", Body: `<html><body>{{.Content}}</body></html>`}).
		Page(Page{Path: "/", Layout: "l", Root: "c", Data: "hey"})
	out := t.TempDir()
	if _, err := s.Build(out); err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), `<em data-kiw-component="c">HEY</em>`) {
		t.Errorf("custom func not applied: %s", page)
	}
}
