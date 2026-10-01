package ssg

import (
	"bytes"
	"path"
	"sort"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// AutoAssetConfig configures automatic CSS/JS injection into every rendered
// document, so layouts never hardcode <link>/<script> tags for site assets.
type AutoAssetConfig struct {
	// Enabled turns injection on. Unset (nil) means enabled — auto-injection is
	// the default so a site ships working asset tags without any template work.
	Enabled *bool `yaml:"enabled"`
	// Exclude lists asset names, base names, or globs that must never be
	// injected (e.g. ["*.min.css"]). Matching is against the registered asset
	// name and its base name.
	Exclude []string `yaml:"exclude"`
	// JSPlacement is where injected scripts go: "head" (default, preserves
	// the {{assetLinks}} behavior where theme scripts run before first paint) or
	// "body" (end of body, for scripts that only need the DOM to exist).
	JSPlacement string `yaml:"js_placement"`
}

// jsInHead reports whether scripts are injected into <head>.
func (c *AutoAssetConfig) jsInHead() bool {
	return c == nil || c.JSPlacement != "body"
}

// isEnabled reports the effective injection flag, defaulting to on.
func (c *AutoAssetConfig) isEnabled() bool {
	return c == nil || c.Enabled == nil || *c.Enabled
}

// autoAssets is the resolved injection state of a Site. Enabled defaults to
// true, so New() sites inject automatically unless AutoAssets(false) opts out.
type autoAssets struct {
	enabled bool
	exclude []string
	// jsInHead injects scripts into <head> (default) so first-paint scripts
	// such as theme bootstrapping still run before the body renders.
	jsInHead bool
}

// skips reports whether an asset is excluded from site-wide injection.
func (a autoAssets) skips(name string) bool {
	base := path.Base(name)
	for _, pat := range a.exclude {
		if pat == "" {
			continue
		}
		if pat == name || pat == base {
			return true
		}
		if ok, err := path.Match(pat, base); err == nil && ok {
			return true
		}
	}
	return false
}

// AutoAssets configures automatic CSS/JS injection. Passing nil keeps the
// default (enabled, scripts in head). This is the pipeline entry point for
// programmatic sites; krewire.yaml drives the same setting via `auto_assets:`.
func (s *Site) AutoAssets(c *AutoAssetConfig) *Site {
	if c == nil {
		s.auto = autoAssets{enabled: true, jsInHead: true}
		return s
	}
	s.auto = autoAssets{
		enabled:  c.isEnabled(),
		exclude:  append([]string(nil), c.Exclude...),
		jsInHead: c.jsInHead(),
	}
	return s
}

// DeclareAsset registers an asset path that is produced outside the SSG build —
// a plugin (e.g. Tailwind) or an external tool writing into the output
// directory. The asset is not written by Build; it is only injected into
// rendered documents so the site links a file it does not own.
func (s *Site) DeclareAsset(names ...string) *Site {
	for _, name := range names {
		name = strings.TrimPrefix(name, "/")
		if name == "" {
			continue
		}
		s.declared[name] = true
	}
	return s
}

// injectAssets rewrites a rendered document so it references every site CSS and
// JS asset exactly once: stylesheets are appended to <head>, and scripts go to
// <head> (the default, so theme scripts still run before first paint) or to the
// end of <body> when `auto_assets.js_placement: body` is set. Assets the
// template already references are left untouched (no duplicates), and
// page-scoped script assets are skipped because injectScripts emits them per
// page.
//
// version is the cache-busting token appended as ?v=<version>.
func (s *Site) injectAssets(doc, version string) (string, error) {
	if !s.auto.enabled {
		return doc, nil
	}
	css, js := s.autoAssetSet()
	if len(css) == 0 && len(js) == 0 {
		return doc, nil
	}
	root, err := html.Parse(strings.NewReader(doc))
	if err != nil {
		return "", err
	}
	referenced := referencedAssets(root)
	head := findElement(root, "head")
	body := findElement(root, "body")
	if head == nil && body == nil {
		return doc, nil
	}
	if head == nil {
		head = body
	}
	suffix := ""
	if version != "" {
		suffix = "?v=" + version
	}
	for _, name := range css {
		url := s.asset(name)
		if referenced[url] || referenced[url+suffix] {
			continue
		}
		head.AppendChild(&html.Node{
			Type:     html.ElementNode,
			Data:     "link",
			DataAtom: atom.Link,
			Attr: []html.Attribute{
				{Key: "rel", Val: "stylesheet"},
				{Key: "href", Val: url + suffix},
			},
		})
	}
	jsTarget := body
	if s.auto.jsInHead {
		jsTarget = head
	}
	for _, name := range js {
		url := s.asset(name)
		if referenced[url] || referenced[url+suffix] {
			continue
		}
		if jsTarget == nil {
			break
		}
		jsTarget.AppendChild(&html.Node{
			Type:     html.ElementNode,
			Data:     "script",
			DataAtom: atom.Script,
			Attr:     []html.Attribute{{Key: "src", Val: url + suffix}},
		})
	}
	var buf bytes.Buffer
	if err := html.Render(&buf, root); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// autoAssetSet returns the CSS and JS assets eligible for site-wide injection.
// Only names under assets/ qualify, because that is the prefix the emitted URL
// (assetURL) and the written output path (writeAssets) agree on; anything else
// (e.g. a stray public/scripts/tailwind.css) is left to the template. Page- and
// layout-scoped scripts and exclude matches are skipped as well.
func (s *Site) autoAssetSet() (css, js []string) {
	names := make([]string, 0, len(s.assets)+len(s.declared))
	for n := range s.assets {
		names = append(names, n)
	}
	for n := range s.declared {
		if _, ok := s.assets[n]; !ok {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		if !strings.HasPrefix(name, "assets/") {
			continue
		}
		if s.scriptAssets[name] || s.auto.skips(name) {
			continue
		}
		switch strings.ToLower(path.Ext(name)) {
		case ".css":
			css = append(css, name)
		case ".js":
			js = append(js, name)
		}
	}
	// The scoped component/layout stylesheet is generated during the build, not
	// registered as an asset, so it is prepended here to stay reachable.
	if s.hasScopedCSS() && !s.auto.skips("assets/style.css") {
		css = append([]string{"assets/style.css"}, css...)
	}
	return css, js
}

// hasScopedCSS reports whether any component or layout carries styles, i.e.
// whether the build will emit assets/style.css.
func (s *Site) hasScopedCSS() bool {
	for _, c := range s.comps {
		if strings.TrimSpace(c.Style) != "" {
			return true
		}
	}
	for _, l := range s.layouts {
		if strings.TrimSpace(l.Style) != "" {
			return true
		}
	}
	return false
}

// referencedAssets collects the href/src URLs of links and scripts already
// present in the document, so injection never duplicates a manual tag.
func referencedAssets(root *html.Node) map[string]bool {
	out := map[string]bool{}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for _, a := range n.Attr {
				switch {
				case n.Data == "link" && a.Key == "href":
				case n.Data == "script" && a.Key == "src":
				default:
					continue
				}
				if v := strings.TrimSpace(a.Val); v != "" {
					out[v] = true
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return out
}

// findElement returns the first element with the given tag name.
func findElement(n *html.Node, name string) *html.Node {
	if n.Type == html.ElementNode && n.Data == name {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findElement(c, name); found != nil {
			return found
		}
	}
	return nil
}

// ScriptAsset registers a script belonging to one layout or page only (a
// <script> block inside a .kiw file). It is emitted as a normal asset but is
// never injected site-wide, because page-scoped scripts are appended by
// injectScripts on the page that declares them.
func (s *Site) ScriptAsset(name, body string) *Site {
	s.assets[name] = body
	s.scriptAssets[name] = true
	return s
}
