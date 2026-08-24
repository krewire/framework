package component

import (
	"context"

	"github.com/krewire/framework/runtime/vdom"
)

type hookKind uint8

const (
	hookState hookKind = iota
	hookRef
	hookMemo
	hookEffect
)

func (k hookKind) String() string {
	switch k {
	case hookState:
		return "UseState"
	case hookRef:
		return "UseRef"
	case hookMemo:
		return "UseMemo"
	default:
		return "UseEffect"
	}
}

type hook struct {
	kind    hookKind
	value   any
	deps    []any
	cleanup func()
}

// Fiber is a component instance: identity, props, hook slots, and the
// previous render tree used for reconciliation.
type Fiber struct {
	Name     string
	Key      string
	Props    Props
	renderer Renderer

	comp    Component
	slots   []hook
	cursor  int
	mounted bool
	prev    *vdom.VNode
	events  map[string]func(context.Context)

	pendingEffects []*hook
}

// NewFiber instantiates name through the registry and returns its fiber.
func NewFiber(name, key string, p Props) (*Fiber, error) {
	factory, err := Lookup(name)
	if err != nil {
		return nil, err
	}
	return &Fiber{Name: name, Key: key, Props: cloneProps(p), comp: factory(cloneProps(p))}, nil
}

// NewFiberWith constructs a fiber around an explicit component instance,
// bypassing the registry.
func NewFiberWith(name string, comp Component) *Fiber {
	return &Fiber{Name: name, comp: comp}
}

// SetRenderer attaches the dirty-notification sink.
func (f *Fiber) SetRenderer(r Renderer) { f.renderer = r }

// Render renders the component with hook machinery active. Hook calls must
// be unconditional and in the same order on every render; violations panic
// with *HookRuleError.
func (f *Fiber) Render() *vdom.VNode {
	beginRender(f)
	tree := f.comp.Render()
	endRender(f)
	return tree
}

// Reconcile applies new props through the ShouldUpdate gate, re-renders, and
// diffs against the previous tree. Nil patches mean the gate skipped the
// update; the first reconcile always renders regardless of the gate.
func (f *Fiber) Reconcile(next Props) []vdom.Patch {
	if f.prev != nil && !f.shouldUpdate(next) {
		return nil
	}
	f.Props = cloneProps(next)
	nextTree := f.Render()
	var patches []vdom.Patch
	if f.prev != nil {
		patches = vdom.Diff(f.prev, nextTree)
	}
	f.prev = nextTree
	if f.mounted {
		f.runPendingEffects()
	}
	return patches
}

// Prev returns the last rendered tree, or nil before the first render.
func (f *Fiber) Prev() *vdom.VNode { return f.prev }

// Seed installs an externally produced tree as the reconciliation base so
// the first Reconcile diffs against server-rendered output instead of
// treating the mount as a fresh render (KWF-T4X9P FRK-WASM-041).
func (f *Fiber) Seed(tree *vdom.VNode) { f.prev = tree }

// FireMount runs OnMount once; subsequent calls are no-ops.
func (f *Fiber) FireMount(ctx context.Context) {
	if f.mounted {
		return
	}
	f.mounted = true
	if m, ok := f.comp.(Mounter); ok {
		m.OnMount(ctx)
	}
	f.runPendingEffects()
}

// FireUnmount runs cleanups newest-first, then OnUnmount.
func (f *Fiber) FireUnmount() {
	for i := len(f.slots) - 1; i >= 0; i-- {
		if c := f.slots[i].cleanup; c != nil {
			c()
			f.slots[i].cleanup = nil
		}
	}
	if u, ok := f.comp.(Unmounter); ok {
		u.OnUnmount()
	}
}

func (f *Fiber) shouldUpdate(next Props) bool {
	su, ok := f.comp.(ShouldUpdater)
	if !ok {
		return true
	}
	return su.ShouldUpdate(next)
}

func (f *Fiber) runPendingEffects() {
	for _, h := range f.pendingEffects {
		if h.cleanup != nil {
			h.cleanup()
			h.cleanup = nil
		}
		if fn, ok := h.value.(func() func()); ok {
			h.cleanup = fn()
		}
	}
	f.pendingEffects = nil
}

func beginRender(f *Fiber) {
	current = f
	f.cursor = 0
	f.events = nil // handlers re-register every render, like hooks slots
}

func endRender(f *Fiber) {
	current = nil
	if f.cursor != len(f.slots) {
		panic(&HookRuleError{Component: f.Name, Slot: f.cursor, Want: "same hook count as previous render", Got: "fewer hooks"})
	}
}

var current *Fiber
