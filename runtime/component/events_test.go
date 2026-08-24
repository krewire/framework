// Tests for KWF-T4X9P
package component

import (
	"context"
	"strings"
	"testing"

	"github.com/krewire/framework/runtime/vdom"
)

// Spec: KWF-T4X9P FRK-WASM-041 Scope: Package
func TestFRK_WASM_041_UseEvent_RegisterLookupAndRefresh(t *testing.T) {
	var calls []string
	handler := func(tag string) func(context.Context) {
		return func(context.Context) { calls = append(calls, tag) }
	}

	useV := "v1"
	comp := &fn{body: func() *vdom.VNode {
		if useV != "" {
			UseEvent("click", "inc", handler(useV))
		}
		return vdom.El("p", nil, vdom.Text("x"))
	}}
	f := NewFiberWith("ev", comp)
	f.Render()

	got := f.Handler("click", "inc")
	if got == nil {
		t.Fatal("handler not registered")
	}
	got(context.Background())
	if len(calls) != 1 || calls[0] != "v1" {
		t.Fatalf("invoke = %v", calls)
	}

	useV = "v2"
	f.Render() // fresh closure replaces the stale one
	calls = nil
	if h := f.Handler("click", "inc"); h == nil {
		t.Fatal("re-registered handler missing")
	} else {
		h(context.Background())
	}
	if len(calls) != 1 || calls[0] != "v2" {
		t.Fatalf("refreshed invoke = %v", calls)
	}

	useV = ""
	f.Render() // drops the registration entirely
	if f.Handler("click", "inc") != nil {
		t.Fatal("unregistered handler must return nil after re-render")
	}
}

// Spec: KWF-T4X9P FRK-WASM-033 Scope: Package
func TestFRK_WASM_033_UseEvent_OutsideRenderPanics(t *testing.T) {
	defer func() {
		if _, ok := recover().(*HookRuleError); !ok {
			t.Fatal("want HookRuleError outside render scope")
		}
	}()
	UseEvent("click", "x", func(context.Context) {})
}

// Spec: KWF-T4X9P FRK-WASM-041 Scope: Package
func TestFRK_WASM_041_UseEvent_EmptyArgsPanicWithDiagnostic(t *testing.T) {
	switcher := true
	comp := &fn{body: func() *vdom.VNode {
		if switcher {
			UseEvent("", "x", func(context.Context) {})
		}
		return vdom.El("p", nil, vdom.Text("x"))
	}}
	f := NewFiberWith("bad", comp)

	switcher = false // first render clean, so slots exist for the panicking one
	f.Render()
	switcher = true

	defer func() {
		r := recover()
		hre, ok := r.(*HookRuleError)
		if !ok || !strings.Contains(hre.Want, "non-empty") {
			t.Fatalf("want diagnostic HookRuleError, got %#v", r)
		}
	}()
	f.Render()
	t.Fatal("expected panic on empty event name")
}
