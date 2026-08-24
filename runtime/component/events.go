package component

import "context"

// Event is the platform-neutral payload delivered to UseEvent handlers.
// The client bridge fills it from the live DOM event (input value, checkbox
// state); tests construct it directly.
type Event struct {
	// Value carries the current text of input-like targets.
	Value string
	// Checked carries checkbox/switch state.
	Checked bool
}

// Handler processes one DOM event. It receives the fiber's ambient context
// plus a neutral payload snapshot.
type Handler func(ctx context.Context, ev Event)

// UseEvent registers fn as the handler identified by event and name for the
// fiber currently rendering — the client counterpart of authoring markup
// like @click="inc", which compiles to data-kiw-on="click:inc". Handlers
// are re-registered on every render (closures stay fresh, mirroring the
// React/Svelte model); the boot layer looks them up after commit via
// Fiber.Handler. Calling outside a render panics with *HookRuleError like
// every other hook (KWF-T4X9P FRK-WASM-033).
func UseEvent(event, name string, fn Handler) {
	f := mustCurrent("UseEvent")
	if event == "" || name == "" || fn == nil {
		panic(&HookRuleError{Component: f.Name, Slot: -1,
			Want: "non-empty event/name and non-nil fn", Got: "empty argument"})
	}
	if f.events == nil {
		f.events = map[string]Handler{}
	}
	f.events[event+" "+name] = fn
}

// Handler returns the live handler for event and name, or nil when this
// render did not register one. Used by the client boot layer to pair
// data-kiw-on attributes with fibers.
func (f *Fiber) Handler(event, name string) Handler {
	return f.events[event+" "+name]
}
