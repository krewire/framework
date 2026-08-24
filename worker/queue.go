package worker

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// Job is the unit of work carried by a queue.
type Job interface {
	Run(ctx context.Context) error
}

// JobID identifies a job within its queue. Cron occurrences keep the ID of
// the logical job they belong to.
type JobID string

// Options configures a single Enqueue call. The zero value is a one-shot,
// default-priority job deliverable immediately and retried with the queue's
// default policy.
type Options struct {
	Priority int
	Delay    time.Duration
	Cron     string
	Retry    *RetryPolicy
}

// DeadLetter is a task that exhausted its retry policy, kept for inspection.
type DeadLetter struct {
	ID       JobID
	Job      Job
	Err      error
	Attempts int
	At       time.Time
}

// Queue is the storage-agnostic job-queue contract (FRK-SVC-060). Retries and
// the dead-letter queue are part of the contract (FRK-SVC-062): Nack consults
// the task policy, requeues with backoff while attempts remain, and files
// exhausted tasks into DLQ. Backends for NATS, Redis, and PostgreSQL conform
// to this same interface (FRK-SVC-063).
type Queue interface {
	// Enqueue stores job for later delivery and returns its ID. An invalid
	// Options.Cron is rejected here.
	Enqueue(ctx context.Context, job Job, opts Options) (JobID, error)
	// Dequeue blocks until a task is deliverable or ctx is done, returning
	// the context error when ctx ends first. A successful Dequeue starts one
	// attempt: exactly one of Ack or Nack must follow.
	Dequeue(ctx context.Context) (*Task, error)
	// Ack resolves a dequeued task as done. For a cron task, Ack re-arms the
	// next occurrence instead of finishing the job.
	Ack(ctx context.Context, task *Task) error
	// Nack returns a dequeued task as failed; the queue applies the task's
	// RetryPolicy, either scheduling the next attempt or dead-lettering it.
	// The task's LastErr becomes the dead-letter reason.
	Nack(ctx context.Context, task *Task) error
	// DLQ returns a snapshot of dead-lettered tasks, oldest first.
	DLQ() []DeadLetter
}

// ErrUnknownTask is returned by Ack and Nack when the task was already
// resolved or never came from this queue.
var ErrUnknownTask = errors.New("worker: unknown task")

// Task is the handle a Queue hands out per delivery. Attempts counts started
// attempts; LastErr records why the current attempt failed and is set by the
// consumer before calling Nack.
type Task struct {
	ID       JobID
	Job      Job
	Options  Options
	Attempts int
	LastErr  error

	runAt time.Time
	seq   uint64
	cron  *Schedule
}

// InMemoryQueue is the default zero-dependency backend (FRK-SVC-063): a
// goroutine-safe priority + delay queue with per-task retries and an
// in-process dead-letter list, usable without network or containers
// (KWF-L5H2F NFR2).
type InMemoryQueue struct {
	mu       sync.Mutex
	pending  []*Task
	inflight map[JobID]*Task
	dlq      []DeadLetter
	seq      atomic.Uint64
	notify   chan struct{}
	now      func() time.Time
}

var _ Queue = (*InMemoryQueue)(nil)

// NewInMemoryQueue returns an empty queue ready for use.
func NewInMemoryQueue() *InMemoryQueue {
	return &InMemoryQueue{
		inflight: make(map[JobID]*Task),
		notify:   make(chan struct{}, 1),
		now:      time.Now,
	}
}

// Enqueue implements Queue. A cron task becomes deliverable immediately;
// subsequent occurrences follow the schedule.
func (q *InMemoryQueue) Enqueue(_ context.Context, job Job, opts Options) (JobID, error) {
	if job == nil {
		return "", errors.New("worker: nil job")
	}
	n := q.seq.Add(1)
	t := &Task{Job: job, Options: opts, seq: n, ID: JobID("job-" + strconv.FormatUint(n, 10))}
	if opts.Cron != "" {
		sched, err := ParseCron(opts.Cron)
		if err != nil {
			return "", err
		}
		t.cron = sched
	}
	t.runAt = q.now().Add(opts.Delay)
	q.mu.Lock()
	q.pending = append(q.pending, t)
	q.mu.Unlock()
	q.wake()
	return t.ID, nil
}

// Dequeue implements Queue, delivering the highest-priority ready task first,
// FIFO among equals, and waiting out delays and cron gaps via ctx.
func (q *InMemoryQueue) Dequeue(ctx context.Context) (*Task, error) {
	for {
		q.mu.Lock()
		task := q.popReady(q.now())
		q.mu.Unlock()
		if task != nil {
			return task, nil
		}

		var tmr *time.Timer
		var timer <-chan time.Time
		if next, ok := q.nextReady(); ok {
			d := next.Sub(q.now())
			if d < 0 {
				d = 0
			}
			tmr = time.NewTimer(d)
			timer = tmr.C
		}
		select {
		case <-ctx.Done():
			if tmr != nil {
				tmr.Stop()
			}
			return nil, ctx.Err()
		case <-timer:
		case <-q.notify:
		}
		if tmr != nil {
			tmr.Stop()
		}
	}
}

// Ack implements Queue. Acking an unknown or already-resolved task returns
// ErrUnknownTask.
func (q *InMemoryQueue) Ack(_ context.Context, task *Task) error {
	if task == nil {
		return ErrUnknownTask
	}
	q.mu.Lock()
	if _, ok := q.inflight[task.ID]; !ok {
		q.mu.Unlock()
		return ErrUnknownTask
	}
	delete(q.inflight, task.ID)
	q.mu.Unlock()
	if task.cron != nil {
		return q.rearm(task)
	}
	return nil
}

// Nack implements Queue: attempts below MaxAttempts requeue the task with the
// policy backoff; the final failure appends a DeadLetter recording LastErr
// and the attempt count.
func (q *InMemoryQueue) Nack(_ context.Context, task *Task) error {
	if task == nil {
		return ErrUnknownTask
	}
	policy := task.Options.Retry.resolved()
	q.mu.Lock()
	if _, ok := q.inflight[task.ID]; !ok {
		q.mu.Unlock()
		return ErrUnknownTask
	}
	delete(q.inflight, task.ID)
	if task.Attempts >= policy.MaxAttempts {
		q.dlq = append(q.dlq, DeadLetter{
			ID:       task.ID,
			Job:      task.Job,
			Err:      task.LastErr,
			Attempts: task.Attempts,
			At:       q.now(),
		})
		q.mu.Unlock()
		return nil
	}
	task.runAt = q.now().Add(policy.Backoff(task.Attempts))
	task.LastErr = nil
	q.pending = append(q.pending, task)
	q.mu.Unlock()
	q.wake()
	return nil
}

// DLQ implements Queue, returning a copy so callers cannot mutate queue state.
func (q *InMemoryQueue) DLQ() []DeadLetter {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]DeadLetter, len(q.dlq))
	copy(out, q.dlq)
	return out
}

func (q *InMemoryQueue) popReady(now time.Time) *Task {
	best := -1
	for i, t := range q.pending {
		if t.runAt.After(now) {
			continue
		}
		if best < 0 || t.Options.Priority > q.pending[best].Options.Priority ||
			(t.Options.Priority == q.pending[best].Options.Priority && t.seq < q.pending[best].seq) {
			best = i
		}
	}
	if best < 0 {
		return nil
	}
	task := q.pending[best]
	q.pending = append(q.pending[:best], q.pending[best+1:]...)
	task.Attempts++
	q.inflight[task.ID] = task
	return task
}

func (q *InMemoryQueue) nextReady() (time.Time, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	var next time.Time
	found := false
	for _, t := range q.pending {
		if !found || t.runAt.Before(next) {
			next, found = t.runAt, true
		}
	}
	return next, found
}

func (q *InMemoryQueue) rearm(task *Task) error {
	next := task.cron.NextFire(task.runAt)
	if next.IsZero() {
		return nil
	}
	occurrence := *task
	occurrence.runAt = next
	occurrence.seq = q.seq.Add(1)
	occurrence.Attempts = 0
	occurrence.LastErr = nil
	q.mu.Lock()
	q.pending = append(q.pending, &occurrence)
	q.mu.Unlock()
	q.wake()
	return nil
}

func (q *InMemoryQueue) wake() {
	select {
	case q.notify <- struct{}{}:
	default:
	}
}
