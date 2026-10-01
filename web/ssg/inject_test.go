// Tests for KWF-AI7Q2 automatic asset injection (FRK-AS-050..073).
// Spec: KWF-AI7Q2. Scope: Framework — Web — SSG — Assets.
package ssg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// bareSite is a site whose layout declares no asset tags at all.
func bareSite() *Site {
	return New().
		Component(Component{
			Name:  "hero",
			Body:  `<h1>{{.}}</h1>`,
			Style: `.hero { color: red; }`,
		}).
		Layout(Layout{
			Name: "base",
			Body: `<!DOCTYPE html><html><head><title>t</title></head><body>{{.Content}}</body></html>`,
		}).
		Page(Page{
			Path:   "/",
			Layout: "base",
			Root:   "hero",
			Data:   map[string]any{"Title": "hi", "Version": "v1.2.3"},
		})
}

func renderIndex(t *testing.T, s *Site) string {
	t.Helper()
	out := t.TempDir()
	if _, err := s.Build(out); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// With no manual tags, every site CSS asset lands in <head> and every site JS
// asset lands in <body>, each with the cache-busting version.
func TestAutoInjectAddsTagsWithoutTemplateTags(t *testing.T) {
	page := renderIndex(t, bareSite().
		Asset("assets/site.css", `body{margin:0}`).
		Asset("assets/site.js", `console.log(1)`))
	for _, want := range []string{
		`<link rel="stylesheet" href="/assets/style.css?v=1.2.3"/>`,
		`<link rel="stylesheet" href="/assets/site.css?v=1.2.3"/>`,
		`<script src="/assets/site.js?v=1.2.3"></script>`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("page missing %s\n%s", want, page)
		}
	}
	if !strings.Contains(page, `</head>`) {
		t.Error("head must stay intact")
	}
}

// Stylesheets always go to <head>; scripts follow auto_assets.js_placement —
// head by default so first-paint scripts still run early, body on request.
func TestAutoInjectPlacesCSSInHeadAndJSInBody(t *testing.T) {
	page := renderIndex(t, bareSite().
		Asset("assets/a.css", `a{color:blue}`).
		Asset("assets/a.js", `void 0`).
		AutoAssets(&AutoAssetConfig{JSPlacement: "body"}))
	headEnd := strings.Index(page, `</head>`)
	linkAt := strings.Index(page, `<link rel="stylesheet" href="/assets/a.css`)
	scriptAt := strings.Index(page, `<script src="/assets/a.js`)
	if linkAt < 0 || scriptAt < 0 {
		t.Fatalf("missing injected tags: %s", page)
	}
	if linkAt > headEnd {
		t.Error("stylesheet must be injected before </head>")
	}
	if scriptAt < headEnd {
		t.Errorf("js_placement: body must inject the script after </head>: %s", page)
	}
}

// The default keeps scripts in <head>, matching the previous assetLinks
// behavior where theme bootstrapping runs before first paint.
func TestAutoInjectDefaultsScriptsToHead(t *testing.T) {
	page := renderIndex(t, bareSite().Asset("assets/a.js", `void 0`))
	headEnd := strings.Index(page, `</head>`)
	scriptAt := strings.Index(page, `<script src="/assets/a.js`)
	if scriptAt < 0 || scriptAt > headEnd {
		t.Errorf("script must default into <head>: %s", page)
	}
}

// A tag the template already emits is never duplicated — matched both with
// and without the cache-busting query string.
func TestAutoInjectSkipsAlreadyReferencedAssets(t *testing.T) {
	for _, tc := range []struct{ name, want string }{
		{"manual-tag-without-version", `<link rel="stylesheet" href="/assets/dup.css">`},
		{"manual-tag-with-version", `<link rel="stylesheet" href="/assets/dup.css?v=1.2.3">`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := New().
				Asset("assets/dup.css", `a{}`).
				Component(Component{Name: "c", Body: `<p>x</p>`}).
				Layout(Layout{
					Name: "base",
					Body: `<!DOCTYPE html><html><head>` + tc.want + `</head><body>{{.Content}}</body></html>`,
				}).
				Page(Page{Path: "/", Layout: "base", Root: "c",
					Data: map[string]any{"Title": "t", "Version": "v1.2.3"}})
			page := renderIndex(t, s)
			if n := strings.Count(page, `href="/assets/dup.css`); n != 1 {
				t.Errorf("dup.css referenced %d times, want 1: %s", n, page)
			}
		})
	}
}

// Page-scoped script assets are never injected site-wide; injectScripts keeps
// emitting them on the page that declares them.
func TestAutoInjectSkipsPageScopedScripts(t *testing.T) {
	s := bareSite().
		Asset("assets/shared.js", `void 0`).
		ScriptAsset("assets/page-only.js", `void 1`)
	s.pages[0].Scripts = []string{"/assets/page-only.js"}
	page := renderIndex(t, s)
	if n := strings.Count(page, `/assets/page-only.js`); n != 1 {
		t.Errorf("page-scoped script referenced %d times, want 1: %s", n, page)
	}
	if strings.Contains(page, `/assets/page-only.js?v=`) {
		t.Errorf("page-scoped script must not be injected site-wide: %s", page)
	}
	if !strings.Contains(page, `<script src="/assets/shared.js?v=1.2.3"></script>`) {
		t.Errorf("shared script not injected: %s", page)
	}
}

// Exclude patterns — full asset name, base name, or glob — opt assets out.
func TestAutoInjectHonorsExclude(t *testing.T) {
	page := renderIndex(t, bareSite().
		Asset("assets/site.css", `a{}`).
		Asset("assets/vendor.min.css", `b{}`).
		Asset("assets/vendor.js", `void 0`).
		AutoAssets(&AutoAssetConfig{Exclude: []string{"vendor.min.css", "*.js"}}))
	if !strings.Contains(page, `href="/assets/site.css`) {
		t.Errorf("non-excluded css must still be injected: %s", page)
	}
	for _, gone := range []string{"vendor.min.css", "vendor.js"} {
		if strings.Contains(page, gone) {
			t.Errorf("%s must be excluded from injection: %s", gone, page)
		}
	}
}

// enabled: false restores fully manual asset tags.
func TestAutoInjectDisabled(t *testing.T) {
	page := renderIndex(t, bareSite().
		Asset("assets/site.css", `a{}`).
		AutoAssets(&AutoAssetConfig{Enabled: boolPtr(false)}))
	if strings.Contains(page, `rel="stylesheet"`) {
		t.Errorf("no stylesheet may be injected when disabled: %s", page)
	}
}

func boolPtr(b bool) *bool { return &b }

// Fingerprinted pipeline output is injected through its manifest URL, not the
// source name.
func TestAutoInjectUsesManifestURL(t *testing.T) {
	page := renderIndex(t, bareSite().
		Asset("assets/site.css", `a{}`).
		Pipeline([]PipelineRule{{Glob: "assets/*.css", Use: []string{"hash"}}}))
	if strings.Contains(page, `href="/assets/site.css?v=`) {
		t.Errorf("fingerprinted asset injected under its source name: %s", page)
	}
	if !strings.Contains(page, `href="/assets/site.`) {
		t.Errorf("fingerprinted asset URL not injected: %s", page)
	}
}

// A site with no styles and no assets renders without any injected tag.
func TestAutoInjectNoAssetsLeavesDocumentClean(t *testing.T) {
	page := renderIndex(t, New().
		Component(Component{Name: "c", Body: `<p>x</p>`}).
		Layout(Layout{Name: "base", Body: `<!DOCTYPE html><html><head></head><body>{{.Content}}</body></html>`}).
		Page(Page{Path: "/", Layout: "base", Root: "c"}))
	if strings.Contains(page, "<link") || strings.Contains(page, "<script") {
		t.Errorf("nothing to inject, document must stay clean: %s", page)
	}
}
