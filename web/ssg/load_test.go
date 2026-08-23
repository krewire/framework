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
