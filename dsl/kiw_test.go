// Tests for KWF-DF3PL
package dsl

import "testing"

func TestParseKiw_FrontmatterAndTemplate(t *testing.T) {
	src := "---\ntitle: Landing\nlayout: Base\n---\n<h1>{{.Title}}</h1>\n<style>h1{color:red}</style>\n<script>console.log(1)</script>"
	mod, err := ParseKiw(src)
	if err != nil {
		t.Fatal(err)
	}
	if mod.Frontmatter["title"] != "Landing" {
		t.Errorf("frontmatter title = %v want Landing", mod.Frontmatter["title"])
	}
	if mod.Body != "<h1>{{.Title}}</h1>" {
		t.Errorf("body = %q want <h1>{{.Title}}</h1>", mod.Body)
	}
	if len(mod.Styles) != 1 || mod.Styles[0] != "h1{color:red}" {
		t.Errorf("styles = %v", mod.Styles)
	}
	if len(mod.Scripts) != 1 || mod.Scripts[0] != "console.log(1)" {
		t.Errorf("scripts = %v", mod.Scripts)
	}
}

func TestParseKiw_NoFrontmatter(t *testing.T) {
	src := "<p>hello</p>"
	mod, err := ParseKiw(src)
	if err != nil {
		t.Fatal(err)
	}
	if mod.Body != "<p>hello</p>" {
		t.Errorf("body = %q", mod.Body)
	}
}

func TestParseKiw_JSParseable(t *testing.T) {
	src := "---\ntitle: Landing\n---\n<div>{{title}}</div>"
	mod, err := ParseKiw(src)
	if err != nil {
		t.Fatal(err)
	}
	if mod.Frontmatter["title"] != "Landing" {
		t.Errorf("frontmatter")
	}
	if mod.Body != "<div>{{title}}</div>" {
		t.Errorf("body")
	}
}
