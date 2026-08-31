package ssg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFingerprint_Deterministic(t *testing.T) {
	a := fingerprint("style.css", []byte("body{}"))
	b := fingerprint("style.css", []byte("body{}"))
	if a != b {
		t.Fatalf("fingerprint not deterministic: %q vs %q", a, b)
	}
	if a != "style."+a[6:14]+".css" {
		t.Fatalf("fingerprint shape wrong: %q", a)
	}
	if fingerprint("style.css", []byte("body{}")) == fingerprint("style.css", []byte("body{x:1}")) {
		t.Fatal("different content must produce different fingerprint")
	}
}

func TestRunPipeline_CopyPassthrough(t *testing.T) {
	res, err := RunPipeline(map[string]string{
		"favicon.ico": "rawbytes",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Assets["favicon.ico"] != "rawbytes" {
		t.Errorf("asset not copied: %q", res.Assets["favicon.ico"])
	}
	if res.Manifest["favicon.ico"] != "/assets/favicon.ico" {
		t.Errorf("manifest = %q", res.Manifest["favicon.ico"])
	}
}

func TestRunPipeline_HashAndMinify(t *testing.T) {
	css := "/* c */  body {  color : red ;  }\n"
	res, err := RunPipeline(map[string]string{
		"assets/style.css": css,
	}, []PipelineRule{
		{Glob: "assets/*.css", Use: []string{"minify-css", "hash"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var found string
	for k, v := range res.Assets {
		if strings.HasPrefix(k, "assets/style.") && strings.HasSuffix(k, ".css") {
			found = k
			if strings.Contains(v, "  ") || strings.Contains(v, "/*") {
				t.Errorf("css not minified: %q", v)
			}
		}
	}
	if found == "" {
		t.Fatal("fingerprinted css not emitted")
	}
	if got := res.Manifest["assets/style.css"]; got != "/assets/"+strings.TrimPrefix(found, "assets/") {
		t.Errorf("manifest = %q, want /assets/%s", got, strings.TrimPrefix(found, "assets/"))
	}
}

func TestRunPipeline_CSSRewritesURLs(t *testing.T) {
	css := ".a{background:url(/assets/logo.png)}"
	res, err := RunPipeline(map[string]string{
		"assets/site.css": css,
		"assets/logo.png": "PNGDATA",
	}, []PipelineRule{
		{Glob: "assets/*.css", Use: []string{"hash"}},
		{Glob: "assets/*.png", Use: []string{"hash"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Find the fingerprinted css body.
	for k, v := range res.Assets {
		if strings.HasPrefix(k, "assets/site.") {
			if !strings.Contains(v, "/assets/logo.") {
				t.Errorf("css url() not rewritten: %q", v)
			}
		}
	}
}

func TestBuild_AssetPipelineAndManifest(t *testing.T) {
	s := New().
		Component(Component{Name: "Root", Body: `<a href="{{asset "assets/style.css"}}">x</a>`}).
		Layout(Layout{Name: "Base", Body: `{{template "Root" .}}`}).
		Page(Page{Path: "/", Title: "Home", Layout: "Base", Root: "Root"}).
		Asset("assets/style.css", "/* c */  a{color:red}").
		Pipeline([]PipelineRule{{Glob: "assets/*.css", Use: []string{"minify-css", "hash"}}})

	out := t.TempDir()
	created, err := s.Build(out)
	if err != nil {
		t.Fatal(err)
	}

	hasManifest := false
	var cssFile string
	for _, c := range created {
		if c == "manifest.json" {
			hasManifest = true
		}
		if strings.HasPrefix(c, "assets/style.") && strings.HasSuffix(c, ".css") {
			cssFile = c
		}
	}
	if !hasManifest {
		t.Fatal("manifest.json not emitted")
	}
	if cssFile == "" {
		t.Fatal("fingerprinted css not written")
	}

	mb, err := os.ReadFile(filepath.Join(out, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mb), "/assets/"+strings.TrimPrefix(cssFile, "assets/")) {
		t.Errorf("manifest missing fingerprinted url:\n%s", mb)
	}
	// The index page must reference the fingerprinted asset URL.
	idx, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(idx), "/assets/"+strings.TrimPrefix(cssFile, "assets/")) {
		t.Errorf("page did not resolve asset() to fingerprinted url:\n%s", idx)
	}
}
