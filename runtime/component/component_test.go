// Tests for KWF-T4X9P
package component

import (
	"context"
	"testing"

	"github.com/krewire/framework/runtime/vdom"
)

type fakeRenderer struct{ calls int }

func (r *fakeRenderer) ScheduleRender(f *Fiber) { r.calls++ }

type fn struct{ body func() *vdom.VNode }

func (c *fn) Render() *vdom.VNode { return c.body() }

// Spec: KWF-T4X9P FRK-WASM-030 Scope: Package
func TestFRK_WASM_030_Fiber_LifecycleHooksAndShouldUpdateGate(t *testing.T) {
	var events []string

	comp := &lifecycleComp{
		render:    func() *vdom.VNode { return vdom.El("p", nil, vdom.Text("x")) },
		onMount:   func(ctx context.Context) { events = append(events, "mount") },
		onUnmount: func() { events = append(events, "unmount") },
		shouldUpdate: func(next Props) bool {
			return next["v"] != "skip"
		},
	}

	f := NewFiberWith("lc", comp)
	if f.Prev() != nil {
		t.Fatal("prev tree must be nil before first render")
	}
	if patches := f.Reconcile(Props{"v": "1"}); patches != nil {
		t.Fatalf("first reconcile renders but has no diff base, got %+v", patches)
	}
	f.FireMount(context.Background())
	f.FireMount(context.Background())

	f.Reconcile(Props{"v": "2"})
	if patches := f.Reconcile(Props{"v": "skip"}); patches != nil {
		t.Fatalf("gate=false must yield nil patches, got %+v", patches)
	}
	f.FireUnmount()

	want := []string{"mount", "unmount"}
	if len(events) != 2 || events[0] != want[0] || events[1] != want[1] {
		t.Fatalf("lifecycle events = %v", events)
	}
	if f.Prev().Tag != "p" {
		t.Fatalf("prev tree not tracked: %+v", f.Prev())
	}
}

// Spec: KWF-T4X9P FRK-WASM-031 Scope: Package
func TestFRK_WASM_031_UseState_PersistsAndSchedulesRender(t *testing.T) {
	r := &fakeRenderer{}
	var set func(int)
	got := -1

	comp := &fn{body: func() *vdom.VNode {
		v, s := UseState(7)
		set = s
		got = v
		return vdom.Text("n")
	}}
	f := NewFiberWith("counter", comp)
	f.SetRenderer(r)

	f.Render()
	if got != 7 || r.calls != 0 {
		t.Fatalf("initial state = %d schedules = %d", got, r.calls)
	}
	set(42)
	if r.calls != 1 {
		t.Fatalf("setter should schedule once, got %d", r.calls)
	}
	f.Render()
	if got != 42 {
		t.Fatalf("state did not persist across render: %d", got)
	}
}

// Spec: KWF-T4X9P FRK-WASM_031 Scope: Package
func TestFRK_WASM_031_UseMemoAndUseRef_StableAcrossRenders(t *testing.T) {
	computes := 0
	var ref *Ref[string]

	comp := &fn{body: func() *vdom.VNode {
		UseMemo(func() int { computes++; return computes }, []any{"k"})
		ref = UseRef("init")
		return vdom.Text("m")
	}}
	f := NewFiberWith("memo", comp)
	for i := 0; i < 3; i++ {
		f.Render()
	}

	if computes != 1 {
		t.Fatalf("UseMemo recomputed %d times with stable deps", computes)
	}
	if ref == nil || ref.Current != "init" {
		t.Fatalf("UseRef identity broken: %+v", ref)
	}
	ref.Current = "changed"
	f.Render()
	if ref.Current != "changed" {
		t.Fatalf("ref lost mutation across renders")
	}
}

// Spec: KWF-T4X9P FRK-WASM_031 Scope: Package
func TestFRK_WASM_031_UseEffect_CommitRunsCleanupAndDepSemantics(t *testing.T) {
	// Use a ref for the effect's state to avoid mutating the dependency
	var runLog Ref[[]string]
	var depRef Ref[[]any]

	effect := func(tag string) func() func() {
		return func() func() {
			log := runLog.Current
			log = append(log, "run:"+tag)
			runLog.Current = log
			return func() {
				log = runLog.Current
				log = append(log, "cleanup:"+tag)
				runLog.Current = log
			}
		}
	}

	// Start with empty dependency
	depRef.Current = []any{}

	comp := &fn{body: func() *vdom.VNode {
		UseEffect(effect("a"), depRef.Current)
		return vdom.Text("e")
	}}
	f := NewFiberWith("eff", comp)
	runLog.Current = []string{}

	f.Render()
	f.FireMount(context.Background())
	if len(runLog.Current) != 1 || runLog.Current[0] != "run:a" {
		t.Fatalf("after mount events = %v", runLog.Current)
	}

	f.Reconcile(Props{}) // same deps slice contents → no re-run
	if len(runLog.Current) != 1 {
		t.Fatalf("stable deps re-ran effect: %v", runLog.Current)
	}

	// Change dependency to trigger re-run (cleanup + new effect)
	depRef.Current = []any{"changed"}
	f.Reconcile(Props{})
	want := []string{"run:a", "cleanup:a", "run:a"}
	if len(runLog.Current) != 3 || runLog.Current[1] != want[1] || runLog.Current[2] != want[2] {
		t.Fatalf("dep-change sequence = %v, want %v", runLog.Current, want)
	}

	f.FireUnmount()
	if runLog.Current[len(runLog.Current)-1] != "cleanup:a" {
		t.Fatalf("unmount must run cleanup: %v", runLog.Current)
	}

	nilDeps := 0
	nd := NewFiberWith("eff-nil", &fn{body: func() *vdom.VNode {
		UseEffect(func() func() { nilDeps++; return nil }, nil)
		return vdom.Text("n")
	}})
	nd.Render()
	nd.FireMount(context.Background())
	nd.Reconcile(Props{})
	if nilDeps != 2 {
		t.Fatalf("nil deps must re-run every commit, ran %d", nilDeps)
	}
}

// Spec: KWF-T4X9P FRK-WASM-033 Scope: Package
func TestFRK_WASM_033_HookRules_ConditionalCallPanics(t *testing.T) {
	switcher := false
	comp := &fn{body: func() *vdom.VNode {
		UseState(1)
		if switcher {
			UseMemo(func() int { return 0 }, []any{})
		}
		return vdom.Text("r")
	}}
	f := NewFiberWith("rules", comp)
	f.Reconcile(Props{}) // First render via Reconcile sets f.prev

	switcher = true
	defer func() {
		r := recover()
		hre, ok := r.(*HookRuleError)
		if !ok {
			t.Fatalf("want *HookRuleError panic, got %#v", r)
		}
		// First render: 1 hook. Second render: 2 hooks -> "more hooks" at slot 1
		if hre.Component != "rules" || hre.Slot != 1 || hre.Want != "same hook count as previous render" || hre.Got != "more hooks" {
			t.Fatalf("diagnostic mismatch: %+v", hre)
		}
	}()
	f.Reconcile(Props{}) // Second render via Reconcile should panic
	t.Fatal("expected panic on conditional hook call")
}

// Spec: KWF-T4X9P FRK-WASM-033 Scope: Package
func TestFRK_WASM_033_HookRules_FewerHooksAndOutsideScopePanics(t *testing.T) {
	dropHook := false
	comp := &fn{body: func() *vdom.VNode {
		UseState(1)
		if !dropHook {
			UseState(2)
		}
		return vdom.Text("x")
	}}
	f := NewFiberWith("fewer", comp)
	f.Render()
	dropHook = true
	func() {
		defer func() {
			if _, ok := recover().(*HookRuleError); !ok {
				t.Fatalf("want HookRuleError for fewer hooks")
			}
		}()
		f.Render()
	}()

	func() {
		defer func() {
			if _, ok := recover().(*HookRuleError); !ok {
				t.Fatalf("want HookRuleError outside render scope")
			}
		}()
		UseState(9)
	}()
}

// Spec: KWF-T4X9P FRK-WASM-032 Scope: Package
func TestFRK_WASM_032_Register_LookupDuplicatesUnknownAndNames(t *testing.T) {
	factory := func(p Props) Component {
		label := p["label"]
		return &fn{body: func() *vdom.VNode { return vdom.El("b", nil, vdom.Text(label)) }}
	}
	if err := Register("reg-test-bold", factory); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := Register("reg-test-bold", factory); err == nil {
		t.Fatal("duplicate register must fail")
	}

	got, err := Lookup("reg-test-bold")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if out := vdom.RenderHTML(got(Props{"label": "hi"}).Render()); out != `<b>hi</b>` {
		t.Fatalf("factory output = %q", out)
	}
	if _, err := Lookup("never-registered"); err == nil {
		t.Fatal("unknown lookup must error")
	}

	found := false
	for _, n := range Names() {
		if n == "reg-test-bold" {
			found = true
		}
	}
	if !found {
		t.Fatal("Names() missing registered component")
	}

	fiber, err := NewFiber("reg-test-bold", "k1", Props{"label": "yo"})
	if err != nil {
		t.Fatalf("NewFiber via registry: %v", err)
	}
	if out := vdom.RenderHTML(fiber.Render()); out != `<b>yo</b>` {
		t.Fatalf("registry-built fiber rendered %q", out)
	}
}

type lifecycleComp struct {
	render       func() *vdom.VNode
	onMount      func(ctx context.Context)
	onUnmount    func()
	shouldUpdate func(next Props) bool
}

func (c *lifecycleComp) Render() *vdom.VNode         { return c.render() }
func (c *lifecycleComp) OnMount(ctx context.Context) { c.onMount(ctx) }
func (c *lifecycleComp) OnUnmount()                  { c.onUnmount() }
func (c *lifecycleComp) ShouldUpdate(next Props) bool {
	return c.shouldUpdate(next)
}
