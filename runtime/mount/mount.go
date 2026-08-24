// Package mount defines the SSR hydration contract shared by the SSG emitter
// and the client boot scanner (KWF-T4X9P FRK-WASM-040/041), using conventional
// lifecycle vocabulary: the server renders HTML, the client mounts onto it,
// and hydrate says when.
//
// A mount point is a server-rendered fragment wrapped in a container carrying
// three attributes:
//
//	<div data-kiw-mount="Counter" data-kiw-hydrate="load"
//	     data-kiw-props='{"initial":0}'>…SSR HTML…</div>
//
// The client runtime scans for these markers, instantiates the registered
// component by name, and attaches behavior without re-rendering SSR text —
// mirroring hydrateRoot semantics from React and createSSRApp().mount() from
// Vue rather than any single framework's branded terms.
package mount

import (
	"fmt"
	"html"
	"strings"

	nethtml "golang.org/x/net/html"
)

// Marker attribute names emitted by Wrap and consumed by Scan.
const (
	AttrMount   = "data-kiw-mount"
	AttrHydrate = "data-kiw-hydrate"
	AttrProps   = "data-kiw-props"
)

// Hydrate says when a mount point attaches client behavior. Values follow
// the common progressive-hydration ladder: immediately, when idle, or when
// visible in the viewport.
type Hydrate string

const (
	// Load hydrates as soon as the client module boots.
	Load Hydrate = "load"
	// Idle hydrates on the first browser idle callback.
	Idle Hydrate = "idle"
	// Visible hydrates when the mount point scrolls into the viewport.
	Visible Hydrate = "visible"
)

// ParseHydrate validates a hydrate value coming from authoring syntax
// (frontmatter `hydrate: load` or script attribute). Unknown values error
// with the valid options listed.
func ParseHydrate(s string) (Hydrate, error) {
	switch Hydrate(s) {
	case Load, Idle, Visible:
		return Hydrate(s), nil
	default:
		return "", fmt.Errorf("mount: unknown hydrate %q (want load|idle|visible)", s)
	}
}

// Mount is one scanned mount point. Props carries the raw JSON payload from
// data-kiw-props; decoding belongs to the component's typed schema.
type Mount struct {
	Name    string
	Hydrate Hydrate
	Props   string
}

// Wrap embeds SSR output in a marked container div. Attribute values are
// HTML-escaped; an empty or null propsJSON omits the props attribute so
// purely static mounts stay minimal.
func Wrap(name string, h Hydrate, propsJSON, innerHTML string) string {
	var b strings.Builder
	b.WriteString(`<div `)
	b.WriteString(AttrMount)
	b.WriteString(`="`)
	b.WriteString(html.EscapeString(name))
	b.WriteString(`" `)
	b.WriteString(AttrHydrate)
	b.WriteString(`="`)
	b.WriteString(string(h))
	b.WriteString(`"`)
	if p := strings.TrimSpace(propsJSON); p != "" && p != "null" {
		b.WriteString(` `)
		b.WriteString(AttrProps)
		b.WriteString(`="`)
		b.WriteString(html.EscapeString(p))
		b.WriteString(`"`)
	}
	b.WriteString(">")
	b.WriteString(innerHTML)
	b.WriteString("</div>")
	return b.String()
}

// Scan extracts every mount point from rendered page HTML in document order,
// including mounts nested inside other mounts. A marked node without a name
// attribute is skipped rather than fatal: partial markers must not break
// page rendering (graceful degradation, FRK-WASM-042).
func Scan(pageHTML string) ([]Mount, error) {
	doc, err := nethtml.Parse(strings.NewReader(pageHTML))
	if err != nil {
		return nil, fmt.Errorf("mount: parse html: %w", err)
	}
	var out []Mount
	var walk func(*nethtml.Node)
	walk = func(n *nethtml.Node) {
		if n.Type == nethtml.ElementNode {
			if name := attr(n, AttrMount); name != "" {
				h := attr(n, AttrHydrate)
				if h == "" {
					h = string(Load)
				}
				out = append(out, Mount{Name: name, Hydrate: Hydrate(h), Props: attr(n, AttrProps)})
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return out, nil
}

func attr(n *nethtml.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
