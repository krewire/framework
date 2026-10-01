package ssg

import (
	"path"
	"strings"
	"testing"
)

// names extracts base names from a URL list for readable assertions.
func names(urls []string) []string {
	out := make([]string, 0, len(urls))
	for _, u := range urls {
		out = append(out, path.Base(u))
	}
	return out
}

// A fully populated site must resolve the canonical cascade order, not
// alphabetical order. This is the regression guard for the original bug, where
// mdbind.css loaded before tailwind.css and therefore could not override it.
func TestInjectedAssetsUsesCascadeOrderNotAlphabetical(t *testing.T) {
	s := New().
		Asset("assets/mdbind.css", "").
		Asset("assets/tailwind.css", "").
		Asset("assets/forge.css", "").
		Asset("assets/theme.css", "").
		Component(Component{Name: "c", Body: "x", Style: ".c{}"})
	css, _ := s.InjectedAssets()
	got := names(css)
	want := []string{"style.css", "tailwind.css", "forge.css", "theme.css", "mdbind.css"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("css order = %v, want %v", got, want)
	}
	if len(got) > 0 && got[0] == "mdbind.css" {
		t.Error("mdbind.css must not load first; it exists to override the site")
	}
}

// A vendor stylesheet registered after the theme must still load before it when
// its layer says so.
func TestInjectedAssetsRespectsExplicitLayer(t *testing.T) {
	s := New().
		AssetWith("assets/print.css", "", WithLayer(LayerScoped)).
		Asset("assets/theme.css", "")
	css, _ := s.InjectedAssets()
	got := names(css)
	if len(got) < 2 {
		t.Fatalf("css = %v, want at least print.css and theme.css", got)
	}
	if got[0] != "print.css" {
		t.Errorf("css order = %v, want print.css first (LayerScoped)", got)
	}
}

// WithOrder breaks ties inside one layer without moving between layers.
func TestInjectedAssetsOrderBreaksTiesWithinLayer(t *testing.T) {
	s := New().
		AssetWith("assets/a.css", "", WithLayer(LayerTheme), WithOrder(2)).
		AssetWith("assets/b.css", "", WithLayer(LayerTheme), WithOrder(1))
	css, _ := s.InjectedAssets()
	got := names(css)
	if len(got) < 2 || got[0] != "b.css" {
		t.Errorf("css order = %v, want b.css before a.css (lower order first)", got)
	}
}

// A plugin can declare its stylesheet and still land in a chosen layer, so
// third-party output does not have to sort by filename.
func TestDeclareAssetWithLayerOrdersPluginOutput(t *testing.T) {
	s := New().
		DeclareAssetWith([]string{"assets/tailwind.css"}, WithLayer(LayerVendor)).
		Asset("assets/theme.css", "")
	css, _ := s.InjectedAssets()
	got := names(css)
	if len(got) != 2 || got[0] != "tailwind.css" || got[1] != "theme.css" {
		t.Errorf("css order = %v, want tailwind.css then theme.css", got)
	}
}

// A builtin that the build never produces must not be linked: a dead <link>
// would ship a 404 into every page.
func TestInjectedAssetsOmitsAbsentBuiltins(t *testing.T) {
	css, js := New().InjectedAssets()
	if len(css) != 0 || len(js) != 0 {
		t.Errorf("empty site plan = css %v js %v, want both empty", css, js)
	}
}

// The scoped stylesheet is generated during the build rather than registered, so
// it is the one builtin that may appear without being registered.
func TestInjectedAssetsIncludesGeneratedScopedCSS(t *testing.T) {
	s := New().Component(Component{Name: "c", Body: "x", Style: ".c{}"})
	css, _ := s.InjectedAssets()
	if len(css) == 0 || path.Base(css[0]) != "style.css" {
		t.Errorf("css = %v, want style.css first", css)
	}
}

func TestParseLayer(t *testing.T) {
	cases := []struct {
		in   string
		want AssetLayer
		ok   bool
	}{
		{"scoped", LayerScoped, true},
		{"base", LayerScoped, true},
		{"VENDOR", LayerVendor, true},
		{"plugin", LayerVendor, true},
		{"component", LayerComponent, true},
		{"forge", LayerComponent, true},
		{"theme", LayerTheme, true},
		{"book", LayerBook, true},
		{"last-mile", LayerPage, true},
		{"last_mile", LayerPage, true},
		{"page", LayerPage, true},
		{"  theme  ", LayerTheme, true},
		{"nope", 0, false},
		{"", 0, false},
	}
	for _, c := range cases {
		got, err := ParseLayer(c.in)
		if c.ok && err != nil {
			t.Errorf("ParseLayer(%q) error = %v", c.in, err)
			continue
		}
		if !c.ok {
			if err == nil {
				t.Errorf("ParseLayer(%q) error = nil, want error", c.in)
			}
			continue
		}
		if got != c.want {
			t.Errorf("ParseLayer(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// A misconfigured layer must be visible without failing the build.
func TestAutoAssetsReportsUnknownLayer(t *testing.T) {
	s := New().AutoAssets(&AutoAssetConfig{
		Order: map[string]AssetOrder{"assets/theme.css": {Layer: "nope"}},
	})
	errs := s.AutoAssetErrors()
	if len(errs) != 1 {
		t.Fatalf("AutoAssetErrors() = %v, want one error", errs)
	}
	if !strings.Contains(errs[0].Error(), "assets/theme.css") {
		t.Errorf("error = %q, want it to name the offending asset", errs[0])
	}
}

// A configured layer must actually move the asset, proving config reaches the plan.
func TestAutoAssetsOrderPinsLayer(t *testing.T) {
	s := New().
		Asset("assets/extra.css", "").
		Asset("assets/theme.css", "").
		AutoAssets(&AutoAssetConfig{
			Order: map[string]AssetOrder{"theme.css": {Layer: "last-mile"}},
		})
	if len(s.AutoAssetErrors()) != 0 {
		t.Fatalf("AutoAssetErrors() = %v, want none", s.AutoAssetErrors())
	}
	css, _ := s.InjectedAssets()
	if len(css) == 0 || path.Base(css[len(css)-1]) != "theme.css" {
		t.Errorf("css = %v, want theme.css last after pinning layer last-mile", css)
	}
}

// The plan must be stable across repeated calls: map iteration order must never
// leak into the output.
func TestInjectedAssetsIsDeterministic(t *testing.T) {
	build := func() []string {
		s := New()
		for _, n := range []string{"assets/z.css", "assets/a.css", "assets/m.css", "assets/b.css"} {
			s.Asset(n, "")
		}
		css, _ := s.InjectedAssets()
		return css
	}
	first := strings.Join(build(), ",")
	for i := 0; i < 25; i++ {
		if got := strings.Join(build(), ","); got != first {
			t.Fatalf("plan changed between runs:\n %s\n %s", first, got)
		}
	}
}

// Exclusions must survive the layered plan.
func TestInjectedAssetsHonorsExclude(t *testing.T) {
	s := New().
		Asset("assets/vendor.min.css", "").
		Asset("assets/theme.css", "").
		AutoAssets(&AutoAssetConfig{Exclude: []string{"*.min.css"}})
	css, _ := s.InjectedAssets()
	for _, u := range css {
		if strings.Contains(u, "vendor.min.css") {
			t.Errorf("css = %v, want vendor.min.css excluded", css)
		}
	}
}

// Page-scoped scripts are emitted per page and must never be site-wide.
func TestInjectedAssetsSkipsPageScopedScripts(t *testing.T) {
	s := New().
		ScriptAsset("assets/layout-Base-0.js", "console.log(1)").
		Asset("assets/app.js", "")
	_, js := s.InjectedAssets()
	for _, u := range js {
		if strings.Contains(u, "layout-Base-0.js") {
			t.Errorf("js = %v, want the page-scoped script excluded", js)
		}
	}
}

// The {{assetLinks}} helper and automatic injection must agree on order; they
// read the same plan, so they cannot drift.
func TestAssetLinksMatchesInjectedOrder(t *testing.T) {
	s := New().
		Asset("assets/mdbind.css", "").
		Asset("assets/tailwind.css", "").
		Asset("assets/app.js", "")
	cssURLs, jsURLs := s.InjectedAssets()
	html := string(s.assetLinks("1.0.0"))

	last := -1
	for _, u := range cssURLs {
		idx := strings.Index(html, u+"?v=1.0.0")
		if idx < 0 {
			t.Fatalf("assetLinks missing %s\ngot: %s", u, html)
		}
		if idx < last {
			t.Errorf("assetLinks order differs from the plan for %s", u)
		}
		last = idx
	}
	for _, u := range jsURLs {
		if !strings.Contains(html, u+"?v=1.0.0") {
			t.Errorf("assetLinks missing %s", u)
		}
	}
}
