//go:build js

package vdom

import (
	"strings"
	"syscall/js"
)

// FromDOM converts a live DOM subtree into a virtual tree so the first
// client render can diff against what the server already emitted instead of
// re-rendering it (KWF-T4X9P FRK-WASM-041). Comment nodes are skipped;
// whitespace text nodes are kept so patch paths stay aligned with SSR HTML.
func FromDOM(root js.Value) *VNode {
	switch root.Get("nodeType").Int() {
	case 3: // text
		return Text(root.Get("nodeValue").String())
	case 1: // element
	default:
		return nil
	}
	n := &VNode{
		Kind:  KindElement,
		Tag:   strings.ToLower(root.Get("tagName").String()),
		Props: map[string]string{},
	}
	attrs := root.Get("attributes")
	for i := 0; i < attrs.Length(); i++ {
		a := attrs.Index(i)
		n.Props[a.Get("name").String()] = a.Get("value").String()
	}
	kids := root.Get("childNodes")
	for i := 0; i < kids.Length(); i++ {
		if c := FromDOM(kids.Index(i)); c != nil {
			n.Children = append(n.Children, c)
		}
	}
	return n
}
