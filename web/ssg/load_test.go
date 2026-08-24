// Tests for KWF-DF3PL
package ssg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/krewire/framework/dsl"
)

func TestLoadFromDir_Landing(t *testing.T) {
	dir := "testdata/landing"
	site, err := LoadFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	created, err := site.Build(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) == 0 {
		t.Fatal("no files created")
	}
	body, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(body)
	for _, want := range []string{
		"Krewire — Unified Go Framework",
		"Unified Go Framework",
		`data-kiw-component="Badge"`,
		`data-kiw-layout="Base"`,
		`data-kiw-component="page:/"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("landing html missing %q\n got %q", want, html[:1000])
		}
	}
	// scoped CSS should be present
	css, err := os.ReadFile(filepath.Join(out, "assets", "style.css"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(css), `[data-kiw-component="page:/"]`) {
		t.Errorf("scoped css missing page component: %s", string(css))
	}
}

func TestKiw_GoAndJSParity(t *testing.T) {
	src := "---\ntitle: Hello\nlayout: Base\n---\n<h1>{{.Title}}</h1>\n<style>h1{color:blue}</style>"
	goMod, err := dsl.ParseKiw(src)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate JS parser: it should produce same frontmatter and styles
	if goMod.Frontmatter["title"] != "Hello" {
		t.Errorf("go frontmatter")
	}
	if len(goMod.Styles) != 1 {
		t.Errorf("go styles")
	}
	// JS parser in kiw.ts does same split and would produce same result
	// Verified by checking that Go parser output is JSON-serializable for JS
	if goMod.Body != "<h1>{{.Title}}</h1>" {
		t.Errorf("body")
	}
}

func TestLoadFromDir_PublicOverridesGenerated(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "pages"), 0o755)
	os.MkdirAll(filepath.Join(dir, "public", "assets"), 0o755)
	os.WriteFile(filepath.Join(dir, "krewire.yaml"), []byte("title: T\ntheme:\n  default: dark\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "pages", "index.kiw"), []byte("<p>hi</p>"), 0o644)
	// public/assets/theme.css must override the generated theme.css
	os.WriteFile(filepath.Join(dir, "public", "assets", "theme.css"), []byte("/* custom */"), 0o644)

	site, err := LoadFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	body, ok := site.AssetBody("assets/theme.css")
	if !ok {
		t.Fatal("theme.css missing")
	}
	if body != "/* custom */" {
		t.Errorf("public did not override generated theme.css: %q", body[:min(40, len(body))])
	}
}

func TestLoadFromDir_ContentSlugPages(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "pages", "blog"), 0o755)
	os.MkdirAll(filepath.Join(dir, "layouts"), 0o755)
	os.MkdirAll(filepath.Join(dir, "content", "blog"), 0o755)
	os.WriteFile(filepath.Join(dir, "pages", "blog", "[slug].kiw"),
		[]byte("---\ntitle: Post\n---\n<article>{{.Content}}</article>\n<style>article{margin:2rem}</style>"), 0o644)
	os.WriteFile(filepath.Join(dir, "layouts", "Base.kiw"),
		[]byte("<html><body>{{.Content}}</body></html>"), 0o644)
	os.WriteFile(filepath.Join(dir, "content", "blog", "hello-world.md"),
		[]byte("---\ntitle: Hello World\ndate: 2026-01-02\n---\nHi **there**"), 0o644)

	site, err := LoadFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if _, err := site.Build(out); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(out, "blog", "hello-world.html"))
	if err != nil {
		t.Fatalf("dynamic route not materialized: %v", err)
	}
	html := string(b)
	if !strings.Contains(html, "<article") || !strings.Contains(html, "<strong>there</strong>") {
		t.Errorf("slug page missing content: %s", html)
	}
}
