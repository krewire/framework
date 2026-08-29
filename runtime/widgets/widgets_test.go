// Tests for KWF-T4X9P
package widgets

import (
	"context"
	"strings"
	"testing"

	"github.com/krewire/framework/runtime/component"
	"github.com/krewire/framework/runtime/vdom"
)

func render(c component.Component) string {
	f := component.NewFiberWith("w", c)
	return vdom.RenderHTML(f.Render())
}

// Spec: KWF-T4X9P FRK-WASM-050 Scope: Unit
func TestFRK_WASM_050_Layout_GoldenHTML(t *testing.T) {
	got := vdom.RenderHTML(Column(Row(Text("a"), SizedBox(10, 20))))
	want := `<div class="kiw-column"><div class="kiw-row">` +
		`<span class="kiw-text">a</span>` +
		`<div class="kiw-sized-box" style="width:10px;height:20px"></div>` +
		`</div></div>`
	if got != want {
		t.Fatalf("layout = %q, want %q", got, want)
	}
	if cls := Container().Props["class"]; cls != "kiw-container" {
		t.Fatalf("container class = %q", cls)
	}
}

// Spec: KWF-T4X9P FRK-WASM-051 Scope: Unit
func TestFRK_WASM_051_Text_ClassOpt(t *testing.T) {
	got := vdom.RenderHTML(Text("hi", ClassOpt("accent")))
	if got != `<span class="kiw-text accent">hi</span>` {
		t.Fatalf("text = %q", got)
	}
}

// Spec: KWF-T4X9P FRK-WASM-052 Scope: Unit
func TestFRK_WASM_052_Button_ClickDispatchesHandler(t *testing.T) {
	var pressed int
	b := Button("+1", func(context.Context, component.Event) { pressed++ })
	f := component.NewFiberWith("btn", b)
	html := vdom.RenderHTML(f.Render())
	if !strings.Contains(html, `data-kiw-on="click:press"`) || !strings.Contains(html, "+1") {
		t.Fatalf("button html = %q", html)
	}
	h := f.Handler("click", "press")
	if h == nil {
		t.Fatal("press handler not registered")
	}
	h(context.Background(), component.Event{})
	h(context.Background(), component.Event{})
	if pressed != 2 {
		t.Fatalf("pressed = %d", pressed)
	}
}

// Spec: KWF-T4X9P FRK-WASM-052 Scope: Unit
func TestFRK_WASM_052_Input_EmitsValueAndDeliversPayload(t *testing.T) {
	var got string
	in := Input("USD", func(v string) { got = v })
	f := component.NewFiberWith("in", in)
	html := vdom.RenderHTML(f.Render())
	if !strings.Contains(html, `value="USD"`) || !strings.Contains(html, `data-kiw-on="input:change"`) {
		t.Fatalf("input html = %q", html)
	}
	f.Handler("input", "change")(context.Background(), component.Event{Value: "IDR"})
	if got != "IDR" {
		t.Fatalf("onInput value = %q", got)
	}
}

// Spec: KWF-T4X9P FRK-WASM-052 Scope: Unit
func TestFRK_WASM_052_CheckboxAndSwitch_CheckedStateAndPayload(t *testing.T) {
	var state bool
	cb := Checkbox(true, "Subscribe", func(b bool) { state = b })
	f := component.NewFiberWith("cb", cb)
	html := vdom.RenderHTML(f.Render())
	if !strings.Contains(html, `type="checkbox"`) || !strings.Contains(html, "<input checked") ||
		!strings.Contains(html, `data-kiw-on="change:toggle"`) || !strings.Contains(html, "Subscribe") {
		t.Fatalf("checkbox html = %q", html)
	}
	f.Handler("change", "toggle")(context.Background(), component.Event{Checked: false})
	if state {
		t.Fatal("unchecked payload not delivered")
	}

	sw := Switch(false, func(bool) {})
	fs := component.NewFiberWith("sw", sw)
	if strings.Contains(vdom.RenderHTML(fs.Render()), `data-state="on"`) {
		t.Fatal("off switch must render data-state=off")
	}
}

// Spec: KWF-T4X9P FRK-WASM-054 Scope: Unit
func TestFRK_WASM_054_ListView_KeysSurviveReconciliation(t *testing.T) {
	list := ListView([]Item{
		{Key: "b", Node: ListTile("Beta", nil, nil)},
		{Key: "a", Node: ListTile("Alpha", nil, nil)},
	})
	old := list
	next := ListView([]Item{
		{Key: "a", Node: ListTile("Alpha", nil, nil)},
		{Key: "b", Node: ListTile("Beta", nil, nil)},
	})
	patches := vdom.Diff(old, next)
	for _, p := range patches {
		if p.Kind == vdom.PatchReplace {
			t.Fatalf("keyed reorder must not replace subtrees: %+v", patches)
		}
	}
	html := vdom.RenderHTML(next)
	if !strings.Contains(html, "kiw-list-item") || !strings.Contains(html, "<li") {
		t.Fatalf("list html = %q", html)
	}
}
