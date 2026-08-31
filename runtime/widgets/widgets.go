// Package widgets is the KWF-T4X9P starter widget catalog: Flutter-shaped
// layout and input primitives rendered as virtual DOM, with client behavior
// wired through component hooks. Static widgets return plain *vdom.VNode;
// stateful ones return component.Component so hooks run inside a fiber.
//
// All interactive markers follow the boot contract: data-kiw-on="event:name"
// pairs with UseEvent registrations of the same event and name.
package widgets

import (
	"context"
	"strconv"

	"github.com/krewire/framework/runtime/component"
	"github.com/krewire/framework/runtime/vdom"
)

// ClassOpt attaches an extra CSS class to any widget.
type ClassOpt string

func applyClass(props map[string]string, opts []any) map[string]string {
	for _, o := range opts {
		if c, ok := o.(ClassOpt); ok {
			props["class"] = joinClass(props["class"], string(c))
		}
	}
	return props
}

func joinClass(a, b string) string {
	if a == "" {
		return b
	}
	return a + " " + b
}

// Text renders a text span (FRK-WASM-051).
func Text(s string, opts ...any) *vdom.VNode {
	return vdom.El("span", applyClass(map[string]string{"class": "kiw-text"}, opts), vdom.Text(s))
}

// Container is a block-level box (FRK-WASM-050).
func Container(children ...*vdom.VNode) *vdom.VNode {
	return vdom.El("div", map[string]string{"class": "kiw-container"}, children...)
}

// Row lays children out horizontally (FRK-WASM-050).
func Row(children ...*vdom.VNode) *vdom.VNode {
	return vdom.El("div", map[string]string{"class": "kiw-row"}, children...)
}

// Column lays children out vertically (FRK-WASM-050).
func Column(children ...*vdom.VNode) *vdom.VNode {
	return vdom.El("div", map[string]string{"class": "kiw-column"}, children...)
}

// SizedBox reserves fixed space in CSS pixels (FRK-WASM-050).
func SizedBox(w, h int) *vdom.VNode {
	style := "width:" + strconv.Itoa(w) + "px;height:" + strconv.Itoa(h) + "px"
	return vdom.El("div", map[string]string{"class": "kiw-sized-box", "style": style})
}

// Stack layers children on top of each other (FRK-WASM-050).
func Stack(children ...*vdom.VNode) *vdom.VNode {
	return vdom.El("div", map[string]string{"class": "kiw-stack"}, children...)
}

// Expanded fills remaining space in a Row or Column (FRK-WASM-050).
func Expanded(child *vdom.VNode) *vdom.VNode {
	return vdom.El("div", map[string]string{"class": "kiw-expanded"}, child)
}

// Image renders a picture (FRK-WASM-051).
func Image(src, alt string, opts ...any) *vdom.VNode {
	return vdom.El("img", applyClass(map[string]string{
		"src": src,
		"alt": alt,
	}, opts))
}

// Icon renders a symbolic glyph (FRK-WASM-051).
func Icon(name string, opts ...any) *vdom.VNode {
	return vdom.El("span", applyClass(map[string]string{
		"class":     "kiw-icon",
		"data-icon": name,
	}, opts), vdom.Text(name))
}

// Button renders a pressable button; onPress fires on click with the neutral
// payload (FRK-WASM-052). It must be instantiated while a fiber is rendering
// because registration uses UseEvent under the hood.
func Button(label string, onPress component.Handler) component.Component {
	return componentFunc(func() *vdom.VNode {
		component.UseEvent("click", "press", onPress)
		return vdom.El("button", map[string]string{
			"type":        "button",
			"class":       "kiw-btn",
			"data-kiw-on": "click:press",
		}, vdom.Text(label))
	})
}

// Input is a controlled text field: value re-renders from the parent, and
// onInput receives each keystroke's current value (FRK-WASM-052).
func Input(value string, onInput func(string)) component.Component {
	return componentFunc(func() *vdom.VNode {
		component.UseEvent("input", "change", func(_ context.Context, ev component.Event) {
			onInput(ev.Value)
		})
		return vdom.El("input", map[string]string{
			"class":       "kiw-input",
			"value":       value,
			"data-kiw-on": "input:change",
		})
	})
}

// Checkbox is a controlled boolean toggle (FRK-WASM-052).
func Checkbox(checked bool, label string, onToggle func(bool)) component.Component {
	return componentFunc(func() *vdom.VNode {
		component.UseEvent("change", "toggle", func(_ context.Context, ev component.Event) {
			onToggle(ev.Checked)
		})
		box := vdom.El("input", map[string]string{
			"type":        "checkbox",
			"class":       "kiw-checkbox",
			"data-kiw-on": "change:toggle",
		})
		if checked {
			box.Props["checked"] = ""
		}
		return wrapLabel(box, label)
	})
}

// Switch is a styled boolean toggle backed by a checkbox (FRK-WASM-052).
func Switch(on bool, onChange func(bool)) component.Component {
	return componentFunc(func() *vdom.VNode {
		component.UseEvent("change", "flip", func(_ context.Context, ev component.Event) {
			onChange(ev.Checked)
		})
		state := "off"
		if on {
			state = "on"
		}
		box := vdom.El("input", map[string]string{
			"type":        "checkbox",
			"class":       "kiw-switch",
			"role":        "switch",
			"data-state":  state,
			"data-kiw-on": "change:flip",
		})
		if on {
			box.Props["checked"] = ""
		}
		return box
	})
}

// ListView renders keyed rows so reordering survives reconciliation
// (FRK-WASM-054); virtualization stays deferred per spec.
type Item struct {
	Key  string
	Node *vdom.VNode
}

func ListView(items []Item) *vdom.VNode {
	kids := make([]*vdom.VNode, len(items))
	for i, it := range items {
		row := it.Node
		row.Key = it.Key // keyed reconciliation (vdom.Diff)
		row.Props["class"] = joinClass(row.Props["class"], "kiw-list-item")
		kids[i] = row
	}
	return vdom.El("ul", map[string]string{"class": "kiw-list-view"}, kids...)
}

// ListTile is the standard ListView row: leading, title, trailing.
func ListTile(title string, leading, trailing *vdom.VNode) *vdom.VNode {
	row := vdom.El("li", map[string]string{"class": "kiw-list-tile"}, vdom.Text(title))
	if leading != nil {
		row.Children = append([]*vdom.VNode{leading}, row.Children...)
	}
	if trailing != nil {
		row.Children = append(row.Children, trailing)
	}
	return row
}

type componentFunc func() *vdom.VNode

func (f componentFunc) Render() *vdom.VNode { return f() }

var _ component.Component = componentFunc(nil)

func wrapLabel(control *vdom.VNode, label string) *vdom.VNode {
	return vdom.El("label", map[string]string{"class": "kiw-field"},
		control, vdom.Text(label))
}
