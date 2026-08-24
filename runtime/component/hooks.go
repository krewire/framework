package component

import "reflect"

// Ref is a stable mutable cell surviving across renders.
type Ref[T any] struct {
	Current T
}

// UseState returns the current state value and a setter that replaces it and
// schedules a re-render through the fiber's Renderer (no-op scheduling when
// none is attached).
func UseState[T any](initial T) (T, func(T)) {
	f := mustCurrent("UseState")
	h := slotFor(f, hookState)
	if h.value == nil {
		h.value = initial
	}
	value := h.value.(T)
	set := func(v T) {
		h.value = v
		if f.renderer != nil {
			f.renderer.ScheduleRender(f)
		}
	}
	return value, set
}

// UseRef returns a stable cell pointer whose Current field persists across
// renders; mutating it never triggers a re-render.
func UseRef[T any](initial T) *Ref[T] {
	h := slotFor(mustCurrent("UseRef"), hookRef)
	if h.value == nil {
		h.value = &Ref[T]{Current: initial}
	}
	return h.value.(*Ref[T])
}

// UseMemo caches fn's result until any dependency changes. Nil deps mean
// recompute every render; an empty non-nil slice means compute once.
func UseMemo[T any](fn func() T, deps []any) T {
	h := slotFor(mustCurrent("UseMemo"), hookMemo)
	if h.value == nil || deps == nil || !depsEqual(h.deps, deps) {
		h.value = fn()
		h.deps = cloneDeps(deps)
	}
	return h.value.(T)
}

// UseEffect queues fn to run after the commit that follows this render. Its
// returned cleanup runs before the effect re-runs and again at unmount.
// Dependency rules match UseMemo: nil re-runs every commit, empty non-nil
// runs exactly once after mount.
func UseEffect(fn func() func(), deps []any) {
	f := mustCurrent("UseEffect")
	h := slotFor(f, hookEffect)
	first := h.value == nil
	changed := deps == nil || !depsEqual(h.deps, deps)
	if first || changed {
		h.value = fn
		h.deps = cloneDeps(deps)
		f.pendingEffects = append(f.pendingEffects, h)
	}
}

func mustCurrent(op string) *Fiber {
	if current == nil {
		panic(&HookRuleError{Component: op, Slot: -1, Want: op, Got: "call outside render"})
	}
	return current
}

func slotFor(f *Fiber, k hookKind) *hook {
	i := f.cursor
	if i < len(f.slots) {
		h := &f.slots[i]
		if h.kind != k {
			panic(&HookRuleError{Component: f.Name, Slot: i, Want: h.kind.String(), Got: k.String()})
		}
		f.cursor++
		return h
	}
	// First render (f.prev == nil) populates slots; subsequent renders must not add new hooks.
	if f.prev != nil {
		panic(&HookRuleError{Component: f.Name, Slot: i, Want: "same hook count as previous render", Got: "more hooks"})
	}
	f.slots = append(f.slots, hook{kind: k})
	f.cursor++
	return &f.slots[i]
}

func depsEqual(a, b []any) bool {
	// Use deep equality for the deps slice contents. This matches React's
	// behavior where dependency arrays are compared by value.
	return reflect.DeepEqual(a, b)
}

func cloneDeps(deps []any) []any {
	out := make([]any, len(deps))
	copy(out, deps)
	return out
}

func cloneProps(p Props) Props {
	out := make(Props, len(p))
	for k, v := range p {
		out[k] = v
	}
	return out
}
