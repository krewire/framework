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

func TestKWF_N4K8Q_GoClientTier_NoNeedJS(t *testing.T) {
	// Spec: KWF-N4K8Q FRK-DSL-030/031 Scope: Unit — Go WASM as primary client, no need to write JS
	src := `<div>hi</div><script lang="go" hydrate="load">var c = $props.initial</script><script lang="ts" hydrate="idle">let x=1</script><script lang="go" server>func Load(){}</script>`
	mod, err := ParseKiw(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(mod.ScriptBlocks) != 3 {
		t.Fatalf("scriptBlocks = %d want 3", len(mod.ScriptBlocks))
	}
	if mod.ScriptBlocks[0].Lang != "go" || mod.ScriptBlocks[0].Hydrate != "load" || mod.ScriptBlocks[0].Server {
		t.Errorf("go client block = %+v", mod.ScriptBlocks[0])
	}
	if mod.ScriptBlocks[1].Lang != "ts" || mod.ScriptBlocks[1].Hydrate != "idle" {
		t.Errorf("ts client block = %+v", mod.ScriptBlocks[1])
	}
	if !mod.ScriptBlocks[2].Server || mod.ScriptBlocks[2].Lang != "go" {
		t.Errorf("go server block = %+v", mod.ScriptBlocks[2])
	}
	// default <script> without lang → js load (FRK-DSL-030)
	mod2, _ := ParseKiw(`<script>console.log(1)</script>`)
	if mod2.ScriptBlocks[0].Lang != "js" || mod2.ScriptBlocks[0].Hydrate != "load" {
		t.Errorf("default script block = %+v", mod2.ScriptBlocks[0])
	}
}

func TestKWF_N4K8Q_StyleScoped(t *testing.T) {
	src := `<style scoped>.btn{color:red}</style><style>h1{}</style>`
	mod, err := ParseKiw(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(mod.StyleBlocks) != 2 {
		t.Fatalf("styleBlocks = %d", len(mod.StyleBlocks))
	}
	if !mod.StyleBlocks[0].Scoped {
		t.Errorf("first style should be scoped")
	}
	if mod.StyleBlocks[1].Scoped {
		t.Errorf("second style should not be scoped")
	}
}
