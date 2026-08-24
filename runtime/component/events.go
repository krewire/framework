package component

import "context"

// UseEvent registers fn as the handler identified by event and name for the
// fiber currently rendering — the client counterpart of authoring markup
// like @click="inc", which compiles to data-kiw-on="click:inc". Handlers
// are re-registered on every render (closures stay fresh, mirroring the
// React/Svelte model) and looked up after commit by the boot layer via
// Fiber.Handler. Calling outside a render panics with *HookRuleError like
// every other hook (KWF-T4X9P FRK-WASM-033).
func UseEvent(event, name string, fn func(ctx context.Context)) {
	f := mustCurrent("UseEvent")
	if event == "" || name == "" || fn == nil {
		panic(&HookRuleError{Component: f.Name, Slot: -1,
			Want: "non-empty event/name and non-nil fn", Got: "empty argument"})
	}
	if f.events == nil {
		f.events = map[string]func(context.Context){}
	}
	f.events[event+" "+name] = fn
}

// Handler returns the live handler for event and name, or nil when this
// render did not register one. Used by the client boot layer to pair
// data-kiw-on attributes with fibers.
func (f *Fiber) Handler(event, name string) func(context.Context) {
	return f.events[event+" "+name]
}
