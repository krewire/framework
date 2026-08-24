//go:build js

package vdom

import "syscall/js"

// Apply transforms the live subtree rooted at root according to patches.
// Path indices address childNodes — matching VNode children because FromDOM
// preserves text nodes — so applying the Diff output front-to-back is safe
// by construction. Attribute updates follow PatchUpdateProps replace-all
// semantics: stale attributes are removed before the new set is applied.
func Apply(root js.Value, patches []Patch) error {
	for _, p := range patches {
		parent, err := resolve(root, p.Path)
		if err != nil {
			return err
		}
		switch p.Kind {
		case PatchRemove:
			parent.Call("removeChild", parent.Get("childNodes").Index(p.Index))
		case PatchInsert:
			ref := js.Null()
			kids := parent.Get("childNodes")
			if p.Index < kids.Length() {
				ref = kids.Index(p.Index)
			}
			parent.Call("insertBefore", create(p.Node), ref)
		case PatchReplace:
			old := nodeAt(parent, p.Index)
			if old.IsUndefined() {
				return pathError{msg: "replace: no child"}
			}
			old.Call("replaceWith", create(p.Node))
		case PatchUpdateText:
			nodeAt(parent, p.Index).Set("nodeValue", p.Text)
		case PatchUpdateProps:
			target := nodeAt(parent, p.Index)
			if target.IsUndefined() {
				return pathError{msg: "props: no child"}
			}
			clearAttrs(target)
			for k, v := range NormalizeProps(p.Props) {
				target.Call("setAttribute", k, v)
			}
		}
	}
	return nil
}

func create(n *VNode) js.Value {
	doc := js.Global().Get("document")
	if n.Kind == KindText {
		return doc.Call("createTextNode", n.Text)
	}
	el := doc.Call("createElement", n.Tag)
	for k, v := range NormalizeProps(n.Props) {
		el.Call("setAttribute", k, v)
	}
	for _, c := range n.Children {
		el.Call("appendChild", create(c))
	}
	return el
}

func resolve(root js.Value, path []int) (js.Value, error) {
	cur := root
	for _, i := range path {
		next := cur.Get("childNodes").Index(i)
		if next.IsUndefined() {
			return js.Value{}, pathError{msg: "resolve: no child"}
		}
		cur = next
	}
	return cur, nil
}

func nodeAt(parent js.Value, idx int) js.Value {
	return parent.Get("childNodes").Index(idx)
}

func clearAttrs(el js.Value) {
	n := el.Get("attributes").Length()
	for i := n - 1; i >= 0; i-- {
		el.Call("removeAttribute", el.Get("attributes").Index(i).Get("name").String())
	}
}

type pathError struct{ msg string }

func (e pathError) Error() string { return "vdom: " + e.msg }
