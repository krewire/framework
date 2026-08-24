// Package schedule batches fiber re-render notifications into per-frame
// flushes so state updates coalesce before reconciliation (KWF-T4X9P
// FRK-WASM-012). On js/wasm the default driver is requestAnimationFrame
// (DefaultFrame); native builds and tests drain deterministically via Flush.
package schedule

import "sync"

// Scheduler deduplicates queued items and delivers each once per frame.
// The zero value is usable only through New; New is required because a
// process callback is what gives the queue meaning.
type Scheduler[T comparable] struct {
	mu       sync.Mutex
	process  func(T)
	frame    func(func())
	pending  []T
	queued   map[T]bool
	frameReq bool
}

// New returns a Scheduler delivering each queued item to process once per
// frame. Attach a driver with Frame; without one, Add enqueues and only an
// explicit Flush drains.
func New[T comparable](process func(T)) *Scheduler[T] {
	return &Scheduler[T]{process: process, queued: make(map[T]bool)}
}

// Frame sets the async driver invoked as frame(flush) on the first Add of a
// batch — requestAnimationFrame in the browser, a test fake elsewhere.
// Passing nil disables async scheduling.
func (s *Scheduler[T]) Frame(fn func(func())) {
	s.mu.Lock()
	s.frame = fn
	s.mu.Unlock()
}

// Add enqueues item. Duplicate adds between frames collapse into one
// delivery; a re-add after flush schedules a fresh delivery.
func (s *Scheduler[T]) Add(item T) {
	s.mu.Lock()
	if s.queued == nil {
		s.queued = make(map[T]bool)
	}
	if s.queued[item] {
		s.mu.Unlock()
		return
	}
	s.queued[item] = true
	s.pending = append(s.pending, item)
	frame, req := s.frame, false
	if frame != nil && !s.frameReq {
		s.frameReq = true
		req = true
	}
	s.mu.Unlock()
	if req {
		frame(s.Flush)
	}
}

// Flush drains every pending item in FIFO order synchronously, handing each
// to process exactly once and clearing dedupe state so future Adds fire.
// Items added while process runs are drained by the following iteration.
func (s *Scheduler[T]) Flush() {
	for {
		s.mu.Lock()
		if len(s.pending) == 0 {
			s.mu.Unlock()
			return
		}
		batch := s.pending
		s.pending = nil
		for k := range s.queued {
			delete(s.queued, k)
		}
		s.frameReq = false
		process := s.process
		s.mu.Unlock()

		if process != nil {
			for _, item := range batch {
				process(item)
			}
		}
	}
}

// Len reports how many items are waiting for the next flush.
func (s *Scheduler[T]) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.pending)
}
