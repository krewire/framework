package ssg

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

// AssetLayer orders site assets so the cascade is deterministic: later layers
// override earlier ones, and scripts in a later layer run after earlier ones.
// The values are ordered, so layers can be sorted without a name lookup.
type AssetLayer int

const (
	// LayerScoped holds the scoped stylesheet generated from component and
	// layout <style> blocks. It is the base layer for site chrome.
	LayerScoped AssetLayer = iota
	// LayerVendor holds third-party output such as a plugin's compiled CSS.
	// Utilities belong here so they can override base component styling.
	LayerVendor
	// LayerComponent holds component-library styling.
	LayerComponent
	// LayerTheme holds theme variables and last-mile design overrides.
	LayerTheme
	// LayerBook holds content-pipeline styling (mdbind) that must win over the
	// site chrome when a book is mounted into the same output.
	LayerBook
	// LayerPage holds per-page overrides and is always last.
	LayerPage
)

// String returns the layer's stable name, as used in configuration.
func (l AssetLayer) String() string {
	switch l {
	case LayerScoped:
		return "scoped"
	case LayerVendor:
		return "vendor"
	case LayerComponent:
		return "component"
	case LayerTheme:
		return "theme"
	case LayerBook:
		return "book"
	case LayerPage:
		return "page"
	default:
		return fmt.Sprintf("layer(%d)", int(l))
	}
}

// ParseLayer resolves a configured layer name. Names are case-insensitive and
// accept "-" or "_" as a separator. An unknown name is an error rather than a
// silent default, because a typo in krewire.yaml would otherwise reorder
// assets invisibly.
func ParseLayer(name string) (AssetLayer, error) {
	switch normalizeLayerName(name) {
	case "scoped", "base":
		return LayerScoped, nil
	case "vendor", "plugin", "tailwind":
		return LayerVendor, nil
	case "component", "components", "forge":
		return LayerComponent, nil
	case "theme":
		return LayerTheme, nil
	case "book", "content", "mdbind":
		return LayerBook, nil
	case "page", "last", "lastmile", "override":
		return LayerPage, nil
	default:
		return 0, fmt.Errorf("ssg: unknown asset layer %q", name)
	}
}

func normalizeLayerName(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	n = strings.ReplaceAll(n, "-", "")
	n = strings.ReplaceAll(n, "_", "")
	return n
}

// AssetKind distinguishes the injectables a site can carry.
type AssetKind uint8

const (
	// AssetCSS is a stylesheet, injected as <link rel="stylesheet">.
	AssetCSS AssetKind = iota
	// AssetJS is a script, injected as <script src>.
	AssetJS
)

// assetMeta is the ordering metadata recorded for one asset name.
type assetMeta struct {
	layer AssetLayer
	order int
	kind  AssetKind
	// scoped marks an asset that belongs to a single layout or page and must
	// never be injected site-wide.
	scoped bool
	// known reports whether the name came from builtinAssets, so a custom
	// registration still inherits that builtin's layer.
	known bool
}

// AssetOrder pins the load position of a single asset from configuration.
type AssetOrder struct {
	// Layer names the cascade layer; empty keeps the asset's default layer.
	Layer string `yaml:"layer"`
	// Order breaks ties inside the layer; lower loads first. It only takes
	// effect together with Layer, since zero is indistinguishable from unset.
	Order int `yaml:"order"`
}

// resolvedAsset is one entry of the final, ordered asset plan.
type resolvedAsset struct {
	name  string
	kind  AssetKind
	layer AssetLayer
	url   string
}

// builtinAsset describes one asset the pipeline knows how to emit. This table
// is the single source of truth for the default load order: injection, the
// {{assetLinks}} helper, and the exported plan all read it, so the order can
// no longer drift between them.
type builtinAsset struct {
	name  string
	kind  AssetKind
	layer AssetLayer
	// order breaks ties inside one layer. Zero means "order by registration
	// sequence", which keeps declaration order stable without extra config.
	order int
	// conditional assets are emitted only when the build actually produces
	// them, so a document never links a file that was never written.
	conditional func(*Site) bool
}

// builtinAssets is the canonical order for a Krewire site: scoped CSS, then
// plugin utilities, then the component library, then theme, then book content
// last so a mounted book wins over the site chrome.
var builtinAssets = []builtinAsset{
	{name: "assets/style.css", kind: AssetCSS, layer: LayerScoped, conditional: func(s *Site) bool { return s.hasScopedCSS() }},
	{name: "assets/tailwind.css", kind: AssetCSS, layer: LayerVendor},
	{name: "assets/forge.css", kind: AssetCSS, layer: LayerComponent},
	{name: "assets/theme.css", kind: AssetCSS, layer: LayerTheme},
	{name: "assets/mdbind.css", kind: AssetCSS, layer: LayerBook},
	{name: "assets/app.js", kind: AssetJS, layer: LayerVendor},
	{name: "assets/docs.js", kind: AssetJS, layer: LayerVendor, order: 1},
}

func builtinByName(name string) (builtinAsset, bool) {
	for _, b := range builtinAssets {
		if b.name == name {
			return b, true
		}
	}
	return builtinAsset{}, false
}

// recordAsset stores ordering metadata for name, allocating a registration
// sequence used as the final tie-break so the output stays deterministic.
func (s *Site) recordAsset(name string, m assetMeta) {
	name = strings.TrimPrefix(strings.TrimSpace(name), "/")
	if name == "" {
		return
	}
	if _, ok := s.assetMeta[name]; !ok {
		s.assetSeq[name] = s.assetSeqNext
		s.assetSeqNext++
	}
	s.assetMeta[name] = m
}

// plan resolves every known asset into its final load order.
func (s *Site) plan() []resolvedAsset {
	seen := map[string]bool{}
	out := make([]resolvedAsset, 0, len(s.assetMeta)+len(builtinAssets))

	// Builtins first, then everything registered or declared, so a custom
	// layer assignment below always wins over a builtin's default.
	collect := func(names []string, builtinPass bool) {
		for _, name := range names {
			if seen[name] {
				continue
			}
			m, ok := s.assetMeta[name]
			if !ok {
				m = defaultMeta(name)
			}
			if b, isBuiltin := builtinByName(name); isBuiltin {
				// A builtin is emitted only when the build actually produces
				// it: registered as an asset, declared by a plugin, or
				// generated by the pipeline. Linking a file that was never
				// written would ship a 404 into every page.
				_, registered := s.assets[name]
				_, declared := s.declared[name]
				generated := b.conditional != nil && b.conditional(s)
				if builtinPass && !registered && !declared && !generated {
					continue
				}
				m.layer = b.layer
				m.kind = b.kind
				m.known = true
			}
			seen[name] = true
			out = append(out, resolvedAsset{name: name, kind: m.kind, layer: m.layer, url: s.asset(name)})
		}
	}

	builtins := make([]string, 0, len(builtinAssets))
	for _, b := range builtinAssets {
		builtins = append(builtins, b.name)
	}
	collect(builtins, true)

	names := make([]string, 0, len(s.assetMeta))
	for name := range s.assetMeta {
		names = append(names, name)
	}
	sort.Strings(names)
	collect(names, false)

	// Total order: layer, then explicit order, then registration sequence, then
	// name. The final key keeps the sort deterministic when everything else ties.
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.layer != b.layer {
			return a.layer < b.layer
		}
		am, aok := s.assetMeta[a.name]
		bm, bok := s.assetMeta[b.name]
		if aok && bok && am.order != bm.order {
			return am.order < bm.order
		}
		if as, bs := s.assetSeq[a.name], s.assetSeq[b.name]; as != bs {
			return as < bs
		}
		return a.name < b.name
	})
	return out
}

// injectedPlan filters the resolved plan down to what injection emits: only
// names under assets/ (the prefix the emitted URL and written path agree on),
// skipping page-scoped assets and exclusions.
func (s *Site) injectedPlan() []resolvedAsset {
	var out []resolvedAsset
	for _, a := range s.plan() {
		if !strings.HasPrefix(a.name, "assets/") {
			continue
		}
		if m, ok := s.assetMeta[a.name]; ok && m.scoped {
			continue
		}
		if s.auto.skips(a.name) {
			continue
		}
		out = append(out, a)
	}
	return out
}

// InjectedAssets returns the ordered CSS and JS URLs that automatic injection
// would link, honoring layers, order, exclusions and page-scoped assets. It is
// the shared plan: the devtool uses it to keep a book mounted in the same
// output styled identically, instead of duplicating the asset list.
func (s *Site) InjectedAssets() (css, js []string) {
	for _, a := range s.injectedPlan() {
		if a.kind == AssetCSS {
			css = append(css, a.url)
		} else {
			js = append(js, a.url)
		}
	}
	return css, js
}

// defaultMeta derives metadata for a name that is not a known builtin.
func defaultMeta(name string) assetMeta {
	m := assetMeta{kind: AssetCSS, layer: LayerComponent}
	switch strings.ToLower(path.Ext(name)) {
	case ".js", ".mjs":
		m.kind = AssetJS
		m.layer = LayerVendor
	}
	return m
}

// assetOption configures how an asset is registered.
type assetOption func(*assetMeta)

// WithLayer places the asset in a cascade layer. Later layers override earlier.
func WithLayer(l AssetLayer) assetOption { return func(m *assetMeta) { m.layer = l } }

// WithOrder breaks ties within a layer; lower loads first.
func WithOrder(n int) assetOption { return func(m *assetMeta) { m.order = n } }
