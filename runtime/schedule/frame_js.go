//go:build js

package schedule

import "syscall/js"

// DefaultFrame returns a driver that schedules fn on the next browser
// animation frame, coalescing all Adds issued during the current frame
// (KWF-T4X9P FRK-WASM-012). The callback releases itself after firing.
func DefaultFrame() func(func()) {
	raf := js.Global().Get("requestAnimationFrame")
	return func(fn func()) {
		var cb js.Func
		cb = js.FuncOf(func(this js.Value, args []js.Value) any {
			defer cb.Release()
			fn()
			return js.Undefined()
		})
		raf.Invoke(cb)
	}
}
