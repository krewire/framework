// Package component provides the shared component model for server-side
// rendering and the future client hydration path (KWF-T4X9P FRK-WASM-030).
//
// A Component renders to a virtual tree (Render() *vdom.VNode); lifecycle
// participation is opt-in via capability interfaces (Mounter, Unmounter,
// ShouldUpdater). Stateful logic uses the generic hooks (UseState, UseRef,
// UseMemo, UseEffect), which bind to the Fiber currently being rendered and
// enforce stable call order.
//
// The package is platform-neutral: it never touches the DOM. A future client
// slice wires Fiber renders into the syscall/js bridge and schedules them
// from a requestAnimationFrame loop; server code can ignore scheduling
// entirely and call Fiber.Render directly.
//
// Fibers are not safe for concurrent use; rendering follows a
// single-threaded, frame-batched model like the DOM it will drive.
package component

import (
	"context"
	"fmt"

	"github.com/krewire/framework/runtime/vdom"
)

// Props seed a component instance. The same string map travels through SSR
// island markers and client hydration, so factories must accept it verbatim.
type Props map[string]string

// Component is anything that renders to a virtual tree.
type Component interface {
	Render() *vdom.VNode
}

// Mounter optionally receives a mount callback after the fiber attaches.
type Mounter interface {
	OnMount(ctx context.Context)
}

// Unmounter optionally receives an unmount callback before teardown ends.
type Unmounter interface {
	OnUnmount()
}

// ShouldUpdater optionally gates prop-driven re-renders. Next carries the
// incoming props; returning false skips the re-render.
type ShouldUpdater interface {
	ShouldUpdate(next Props) bool
}

// Factory builds a component instance from island props.
type Factory func(Props) Component

// Renderer is notified when hook setters mark a fiber dirty. The client
// runtime implements it over a frame scheduler; tests use counters.
type Renderer interface {
	ScheduleRender(f *Fiber)
}

// HookRuleError panics out of a render when the Rules of Hooks are violated
// (a hook called conditionally, or fewer hooks than the previous render).
type HookRuleError struct {
	Component string
	Slot      int
	Want      string
	Got       string
}

func (e *HookRuleError) Error() string {
	return fmt.Sprintf("component %q: hook rule violated at slot %d: want %s, got %s",
		e.Component, e.Slot, e.Want, e.Got)
}
