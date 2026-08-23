package app

import (
	"sync"
	"time"
)

// recordingTracer records before/after resolve hooks for verification.
type recordingTracer struct {
	mu    sync.Mutex
	order []string
}

func (r *recordingTracer) BeforeResolve(typeName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.order = append(r.order, "before:"+typeName)
}

func (r *recordingTracer) AfterResolve(typeName string, _ time.Duration, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.order = append(r.order, "after:"+typeName)
	if err != nil {
		r.order = append(r.order, "error:"+typeName)
	}
}

func (r *recordingTracer) events() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.order))
	copy(out, r.order)
	return out
}
