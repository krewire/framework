//go:build js

package boot

import (
	"context"
	"syscall/js"
	"time"

	"github.com/krewire/framework/runtime/component"
	"github.com/krewire/framework/runtime/mount"
	"github.com/krewire/framework/runtime/schedule"
	"github.com/krewire/framework/runtime/vdom"
)

// queue coalesces state-driven re-renders of every mounted fiber into one
// animation frame (FRK-WASM-012).
var queue = schedule.New[*job](func(j *job) { j.paint() })

// Hydrate wires every mount point in the document to its registered
// component. Failures are isolated per mount: one broken mount logs a
// warning and the rest still come alive (FRK-WASM-042/043). Safe to call
// once per page load.
func Hydrate() {
	doc := js.Global().Get("document")
	nodes := doc.Call("querySelectorAll", "["+mount.AttrMount+"]")
	queue.Frame(schedule.DefaultFrame())

	for i := 0; i < nodes.Length(); i++ {
		el := nodes.Index(i)
		name := attrOr(el, mount.AttrMount, "")
		if name == "" {
			continue // partial marker: skip gracefully
		}
		j, err := newJob(el, name)
		if err != nil {
			warn(name, err)
			continue
		}
		h := mount.Hydrate(attrOr(el, mount.AttrHydrate, string(mount.Load)))
		scheduleAt(h, el, func() {
			if err := j.attach(); err != nil {
				warn(name, err)
				return
			}
			j.fiber.FireMount(context.Background())
			j.bindEvents()
		})
	}
}

type job struct {
	el       js.Value
	name     string
	props    component.Props
	fiber    *component.Fiber
	handlers map[string]component.Handler
}

func newJob(el js.Value, name string) (*job, error) {
	var payload []byte
	if p := attrOr(el, mount.AttrProps, ""); p != "" {
		payload = []byte(p)
	}
	props, err := FlattenProps(payload)
	if err != nil {
		return nil, err
	}
	fiber, err := component.NewFiber(name, attrOr(el, "data-kiw-key", ""), props)
	if err != nil {
		return nil, err
	}
	j := &job{el: el, name: name, props: props, fiber: fiber}
	j.fiber.SetRenderer(rendererFunc(func() { queue.Add(j) }))
	return j, nil
}

// attach seeds the fiber with the live DOM and applies the first diff,
// which for parity-clean SSR is empty — listeners attach without touching
// text nodes (FRK-WASM-041 text parity).
func (j *job) attach() error {
	tree := vdom.FromDOM(firstElement(j.el))
	if tree == nil {
		return nil
	}
	j.fiber.Seed(tree)
	return vdom.Apply(j.el, j.fiber.Reconcile(j.props))
}

func (j *job) paint() {
	if err := vdom.Apply(j.el, j.fiber.Reconcile(j.props)); err != nil {
		warn(j.name, err)
	}
}

func attrOr(el js.Value, key, def string) string {
	a := el.Get("attributes").Call("getNamedItem", key)
	if a.IsNull() || a.IsUndefined() {
		return def
	}
	return a.Get("value").String()
}

func firstElement(container js.Value) js.Value {
	kids := container.Get("children")
	if kids.Length() == 0 {
		return container
	}
	return kids.Index(0)
}

type rendererFunc func()

func (f rendererFunc) ScheduleRender(*component.Fiber) { f() }

var _ component.Renderer = rendererFunc(nil)

func warn(name string, err error) {
	js.Global().Get("console").Call("warn", "[kiw] mount "+name+": "+err.Error())
}

// scheduleAt honors the progressive-hydration ladder with conventional
// browser primitives: immediate, idle callback, or viewport intersection.
func scheduleAt(h mount.Hydrate, el js.Value, fn func()) {
	switch h {
	case mount.Idle:
		if ic := js.Global().Get("requestIdleCallback"); !ic.IsUndefined() {
			invokeOnce(ic, fn)
			return
		}
		time.AfterFunc(0, fn)
	case mount.Visible:
		if io := js.Global().Get("IntersectionObserver"); !io.IsUndefined() {
			var cb js.Func
			cb = js.FuncOf(func(this js.Value, args []js.Value) any {
				if args[0].Index(0).Get("isIntersecting").Bool() {
					fn()
					cb.Release()
				}
				return js.Undefined()
			})
			io.New(cb).Call("observe", el)
			return
		}
		fn()
	default: // mount.Load and unrecognized values degrade to immediate
		fn()
	}
}

func invokeOnce(f js.Value, fn func()) {
	var cb js.Func
	cb = js.FuncOf(func(this js.Value, args []js.Value) any {
		cb.Release()
		fn()
		return js.Undefined()
	})
	f.Invoke(cb)
}
