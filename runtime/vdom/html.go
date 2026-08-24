package vdom

import (
	"html"
	"sort"
	"strings"
)

// voidElements render without a closing tag and must not carry children.
var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true,
	"hr": true, "img": true, "input": true, "link": true, "meta": true,
	"source": true, "track": true, "wbr": true,
}

// RenderHTML renders the tree to an HTML string using the same normalized
// props as Diff, guaranteeing SSR output and client patches agree. Text and
// attribute values are escaped.
func RenderHTML(tree *VNode) string {
	var b strings.Builder
	renderNode(&b, tree)
	return b.String()
}

func renderNode(b *strings.Builder, n *VNode) {
	if n == nil {
		return
	}
	switch n.Kind {
	case KindText:
		b.WriteString(html.EscapeString(n.Text))
	case KindComponent:
		b.WriteString(`<kiw-component name="`)
		b.WriteString(html.EscapeString(n.ComponentName))
		b.WriteString(`"`)
		if n.Key != "" {
			b.WriteString(` data-kiw-key="`)
			b.WriteString(html.EscapeString(n.Key))
			b.WriteString(`"`)
		}
		propsAttrs(b, NormalizeProps(n.Props))
		b.WriteString(">")
		for _, c := range n.Children {
			renderNode(b, c)
		}
		b.WriteString(`</kiw-component>`)
	default:
		b.WriteString("<")
		b.WriteString(n.Tag)
		propsAttrs(b, NormalizeProps(n.Props))
		if voidElements[n.Tag] {
			b.WriteString(">")
			return
		}
		b.WriteString(">")
		for _, c := range n.Children {
			renderNode(b, c)
		}
		b.WriteString("</")
		b.WriteString(n.Tag)
		b.WriteString(">")
	}
}

func propsAttrs(b *strings.Builder, props map[string]string) {
	for _, k := range sortedKeys(props) {
		v := props[k]
		b.WriteString(" ")
		b.WriteString(k)
		if v == "" && BoolAttrs[k] {
			continue
		}
		b.WriteString(`="`)
		b.WriteString(html.EscapeString(v))
		b.WriteString(`"`)
	}
}

func sortedKeys(props map[string]string) []string {
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
