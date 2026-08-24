//go:build js

package boot

import (
	"context"
	"strconv"
	"strings"
	"syscall/js"

	"github.com/krewire/framework/runtime/component"
)

const (
	attrOn    = "data-kiw-on"
	jobIDAttr = "data-kiw-job"
)

var (
	jobs     = map[string]*job{}
	jobSeq   int
	listened = map[string]bool{}
)

// bindEvents pairs every data-kiw-on="type:name" marker inside the mount
// with the handler its fiber registered this render. One delegated listener
// per event type serves the whole document; dispatch resolves through
// closest(), so nodes added by later renders work without rebinding.
func (j *job) bindEvents() {
	marked := j.el.Call("querySelectorAll", "["+attrOn+"]")
	for i := 0; i < marked.Length(); i++ {
		el := marked.Index(i)
		spec := el.Get("attributes").Call("getNamedItem", attrOn)
		if spec.IsNull() {
			continue
		}
		eventType, name, ok := strings.Cut(spec.Get("value").String(), ":")
		if !ok {
			continue
		}
		fn := j.fiber.Handler(eventType, name)
		if fn == nil {
			warn(j.name, errSilent{"handler " + eventType + ":" + name + " not registered"})
			continue
		}
		if j.handlers == nil {
			j.handlers = map[string]component.Handler{}
		}
		j.handlers[eventType+" "+name] = fn
		installListener(eventType)
	}
	if len(j.handlers) > 0 && attrOr(j.el, jobIDAttr, "") == "" {
		jobSeq++
		id := strconv.Itoa(jobSeq)
		jobs[id] = j
		j.el.Call("setAttribute", jobIDAttr, id)
	}
}

type errSilent struct{ msg string }

func (e errSilent) Error() string { return e.msg }

// eventPayload snapshots the neutral fields the widget kit relies on.
func eventPayload(ev js.Value) component.Event {
	t := ev.Get("target")
	checked := false
	if c := t.Get("checked"); !c.IsUndefined() {
		checked = c.Bool()
	}
	v := ""
	if val := t.Get("value"); !val.IsUndefined() {
		v = val.String()
	}
	return component.Event{Value: v, Checked: checked}
}

func installListener(eventType string) {
	if listened[eventType] {
		return
	}
	listened[eventType] = true
	var cb js.Func
	cb = js.FuncOf(func(this js.Value, args []js.Value) any {
		dispatch(eventType, args[0])
		return js.Undefined()
	})
	js.Global().Get("document").Call("addEventListener", eventType, cb)
}

// dispatch resolves event.target → nearest [data-kiw-on] marker → its mount
// owner, then invokes the matching handler. Unknown pairings are ignored:
// one stray click must never break other mounts.
func dispatch(eventType string, ev js.Value) {
	target := ev.Get("target")
	marker := target.Call("closest", "["+attrOn+"]")
	if marker.IsNull() || marker.IsUndefined() {
		return
	}
	spec := marker.Get("attributes").Call("getNamedItem", attrOn)
	if spec.IsNull() {
		return
	}
	eventName, name, ok := strings.Cut(spec.Get("value").String(), ":")
	if !ok || eventName != eventType {
		return
	}
	root := target.Call("closest", "["+jobIDAttr+"]")
	if root.IsNull() || root.IsUndefined() {
		return
	}
	j := jobs[attrOr(root, jobIDAttr, "")]
	if j == nil {
		return
	}
	if fn := j.handlers[eventType+" "+name]; fn != nil {
		fn(context.Background(), eventPayload(ev))
	}
}
