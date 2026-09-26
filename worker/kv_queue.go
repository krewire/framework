package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/krewire/framework/storage"
)

// SerializableJob is a Job that can be persisted to and reconstructed from storage.
type SerializableJob interface {
	Job
	// JobType returns the registered type name for the job.
	JobType() string
	// Encode serializes the job's internal payload into bytes.
	Encode() ([]byte, error)
	// Decode restores the job's internal state from bytes.
	Decode([]byte) error
}

// JobFactory instantiates a new instance of a registered job type.
type JobFactory func() SerializableJob

// JobRegistry maps job type names to their constructor factories.
type JobRegistry struct {
	mu        sync.RWMutex
	factories map[string]JobFactory
}

// NewJobRegistry creates an empty JobRegistry.
func NewJobRegistry() *JobRegistry {
	return &JobRegistry{
		factories: make(map[string]JobFactory),
	}
}

// Register registers a named job factory.
func (r *JobRegistry) Register(name string, factory JobFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[name] = factory
}

// Resolve creates a new Job instance from a registered type name and payload.
func (r *JobRegistry) Resolve(name string, payload []byte) (Job, error) {
	r.mu.RLock()
	factory, ok := r.factories[name]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("worker: unregistered job type %q", name)
	}
	job := factory()
	if err := job.Decode(payload); err != nil {
		return nil, fmt.Errorf("worker: decode job %q: %w", name, err)
	}
	return job, nil
}

// PayloadJob is a generic SerializableJob helper for byte or JSON payloads.
type PayloadJob struct {
	Type    string
	Data    []byte
	Handler func(ctx context.Context, payload []byte) error `json:"-"`
}

func (p *PayloadJob) JobType() string                 { return p.Type }
func (p *PayloadJob) Encode() ([]byte, error)        { return p.Data, nil }
func (p *PayloadJob) Decode(b []byte) error          { p.Data = b; return nil }
func (p *PayloadJob) Run(ctx context.Context) error {
	if p.Handler != nil {
		return p.Handler(ctx, p.Data)
	}
	return nil
}

// taskRecord is the JSON-serializable representation of a Task in KV storage.
type taskRecord struct {
	ID       JobID         `json:"id"`
	Type     string        `json:"type"`
	Payload  []byte        `json:"payload,omitempty"`
	Priority int           `json:"priority"`
	Delay    time.Duration `json:"delay"`
	Cron     string        `json:"cron,omitempty"`
	Attempts int           `json:"attempts"`
	RunAt    time.Time     `json:"run_at"`
	Seq      uint64        `json:"seq"`
}

type deadLetterRecord struct {
	ID       JobID     `json:"id"`
	Type     string    `json:"type"`
	Payload  []byte    `json:"payload,omitempty"`
	Err      string    `json:"err"`
	Attempts int       `json:"attempts"`
	At       time.Time `json:"at"`
}

// KVQueue is a persistent, durable job queue backed by storage.KV.
type KVQueue struct {
	kv       storage.KV
	registry *JobRegistry
	mu       sync.Mutex
	pending  []*Task
	inflight map[JobID]*Task
	seq      atomic.Uint64
	notify   chan struct{}
	now      func() time.Time
}

var _ Queue = (*KVQueue)(nil)

// KVQueueOption configures a KVQueue.
type KVQueueOption func(*KVQueue)

// WithJobRegistry attaches a JobRegistry to KVQueue for restoring jobs on startup.
func WithJobRegistry(reg *JobRegistry) KVQueueOption {
	return func(q *KVQueue) {
		q.registry = reg
	}
}

// NewKVQueue creates a persistent queue backed by storage.KV and restores any existing tasks.
func NewKVQueue(ctx context.Context, kv storage.KV, opts ...KVQueueOption) (*KVQueue, error) {
	if kv == nil {
		return nil, errors.New("worker: nil storage.KV backend")
	}
	q := &KVQueue{
		kv:       kv,
		registry: NewJobRegistry(),
		inflight: make(map[JobID]*Task),
		notify:   make(chan struct{}, 1),
		now:      time.Now,
	}
	for _, opt := range opts {
		opt(q)
	}

	if err := q.restore(ctx); err != nil {
		return nil, fmt.Errorf("worker: restore queue from kv: %w", err)
	}
	return q, nil
}

func (q *KVQueue) restore(ctx context.Context) error {
	keys, err := q.kv.List(ctx, "worker/tasks/")
	if err != nil {
		return err
	}

	var maxSeq uint64
	for _, k := range keys {
		val, ok, err := q.kv.Get(ctx, k)
		if err != nil || !ok {
			continue
		}
		var rec taskRecord
		if err := json.Unmarshal(val, &rec); err != nil {
			continue
		}

		job, err := q.registry.Resolve(rec.Type, rec.Payload)
		if err != nil {
			// If job cannot be resolved, create a placeholder payload job
			job = &PayloadJob{Type: rec.Type, Data: rec.Payload}
		}

		task := &Task{
			ID:       rec.ID,
			Job:      job,
			Attempts: rec.Attempts,
			Options: Options{
				Priority: rec.Priority,
				Delay:    rec.Delay,
				Cron:     rec.Cron,
			},
			runAt: rec.RunAt,
			seq:   rec.Seq,
		}
		if rec.Cron != "" {
			if sched, err := ParseCron(rec.Cron); err == nil {
				task.cron = sched
			}
		}

		if rec.Seq > maxSeq {
			maxSeq = rec.Seq
		}
		q.pending = append(q.pending, task)
	}
	q.seq.Store(maxSeq)
	return nil
}

// Enqueue persists a job to KV storage and enqueues it for execution.
func (q *KVQueue) Enqueue(ctx context.Context, job Job, opts Options) (JobID, error) {
	if job == nil {
		return "", errors.New("worker: nil job")
	}
	n := q.seq.Add(1)
	id := JobID("job-" + strconv.FormatUint(n, 10))
	t := &Task{Job: job, Options: opts, seq: n, ID: id}

	if opts.Cron != "" {
		sched, err := ParseCron(opts.Cron)
		if err != nil {
			return "", err
		}
		t.cron = sched
	}
	t.runAt = q.now().Add(opts.Delay)

	// Persist task to KV
	jobType := "generic"
	var payload []byte
	if sj, ok := job.(SerializableJob); ok {
		jobType = sj.JobType()
		var err error
		payload, err = sj.Encode()
		if err != nil {
			return "", fmt.Errorf("worker: encode job: %w", err)
		}
	}

	rec := taskRecord{
		ID:       t.ID,
		Type:     jobType,
		Payload:  payload,
		Priority: opts.Priority,
		Delay:    opts.Delay,
		Cron:     opts.Cron,
		Attempts: t.Attempts,
		RunAt:    t.runAt,
		Seq:      t.seq,
	}
	b, err := json.Marshal(rec)
	if err != nil {
		return "", err
	}
	if err := q.kv.Put(ctx, "worker/tasks/"+string(id), b); err != nil {
		return "", fmt.Errorf("worker: persist task: %w", err)
	}

	q.mu.Lock()
	q.pending = append(q.pending, t)
	q.mu.Unlock()
	q.wake()
	return t.ID, nil
}

// Dequeue delivers the highest priority ready task from persistent queue.
func (q *KVQueue) Dequeue(ctx context.Context) (*Task, error) {
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

// Ack removes the finished task from KV storage or rearms the next cron iteration.
func (q *KVQueue) Ack(ctx context.Context, task *Task) error {
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
		return q.rearm(ctx, task)
	}

	// Delete completed task from persistent storage
	_ = q.kv.Delete(ctx, "worker/tasks/"+string(task.ID))
	return nil
}

// Nack handles task failure, applying retry backoff or filing to DLQ in persistent storage.
func (q *KVQueue) Nack(ctx context.Context, task *Task) error {
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
		q.mu.Unlock()
		// Remove from active tasks and write to DLQ in KV
		_ = q.kv.Delete(ctx, "worker/tasks/"+string(task.ID))
		jobType := "generic"
		var payload []byte
		if sj, ok := task.Job.(SerializableJob); ok {
			jobType = sj.JobType()
			payload, _ = sj.Encode()
		}
		errStr := ""
		if task.LastErr != nil {
			errStr = task.LastErr.Error()
		}
		dlqRec := deadLetterRecord{
			ID:       task.ID,
			Type:     jobType,
			Payload:  payload,
			Err:      errStr,
			Attempts: task.Attempts,
			At:       q.now(),
		}
		b, _ := json.Marshal(dlqRec)
		_ = q.kv.Put(ctx, "worker/dlq/"+string(task.ID), b)
		return nil
	}

	task.runAt = q.now().Add(policy.Backoff(task.Attempts))
	task.LastErr = nil
	q.pending = append(q.pending, task)
	q.mu.Unlock()

	// Update task state in KV
	jobType := "generic"
	var payload []byte
	if sj, ok := task.Job.(SerializableJob); ok {
		jobType = sj.JobType()
		payload, _ = sj.Encode()
	}
	rec := taskRecord{
		ID:       task.ID,
		Type:     jobType,
		Payload:  payload,
		Priority: task.Options.Priority,
		Delay:    task.Options.Delay,
		Cron:     task.Options.Cron,
		Attempts: task.Attempts,
		RunAt:    task.runAt,
		Seq:      task.seq,
	}
	b, _ := json.Marshal(rec)
	_ = q.kv.Put(ctx, "worker/tasks/"+string(task.ID), b)

	q.wake()
	return nil
}

// DLQ returns all dead-lettered tasks recorded in persistent storage.
func (q *KVQueue) DLQ() []DeadLetter {
	ctx := context.Background()
	keys, err := q.kv.List(ctx, "worker/dlq/")
	if err != nil {
		return nil
	}
	var out []DeadLetter
	for _, k := range keys {
		val, ok, err := q.kv.Get(ctx, k)
		if err != nil || !ok {
			continue
		}
		var rec deadLetterRecord
		if err := json.Unmarshal(val, &rec); err != nil {
			continue
		}
		job, err := q.registry.Resolve(rec.Type, rec.Payload)
		if err != nil {
			job = &PayloadJob{Type: rec.Type, Data: rec.Payload}
		}
		var retErr error
		if rec.Err != "" {
			retErr = errors.New(rec.Err)
		}
		out = append(out, DeadLetter{
			ID:       rec.ID,
			Job:      job,
			Err:      retErr,
			Attempts: rec.Attempts,
			At:       rec.At,
		})
	}
	return out
}

func (q *KVQueue) popReady(now time.Time) *Task {
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

func (q *KVQueue) nextReady() (time.Time, bool) {
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

func (q *KVQueue) rearm(ctx context.Context, task *Task) error {
	next := task.cron.NextFire(task.runAt)
	if next.IsZero() {
		_ = q.kv.Delete(ctx, "worker/tasks/"+string(task.ID))
		return nil
	}
	occurrence := *task
	occurrence.runAt = next
	occurrence.seq = q.seq.Add(1)
	occurrence.Attempts = 0
	occurrence.LastErr = nil

	jobType := "generic"
	var payload []byte
	if sj, ok := task.Job.(SerializableJob); ok {
		jobType = sj.JobType()
		payload, _ = sj.Encode()
	}
	rec := taskRecord{
		ID:       occurrence.ID,
		Type:     jobType,
		Payload:  payload,
		Priority: occurrence.Options.Priority,
		Delay:    occurrence.Options.Delay,
		Cron:     occurrence.Options.Cron,
		Attempts: occurrence.Attempts,
		RunAt:    occurrence.runAt,
		Seq:      occurrence.seq,
	}
	b, _ := json.Marshal(rec)
	_ = q.kv.Put(ctx, "worker/tasks/"+string(occurrence.ID), b)

	q.mu.Lock()
	q.pending = append(q.pending, &occurrence)
	q.mu.Unlock()
	q.wake()
	return nil
}

func (q *KVQueue) wake() {
	select {
	case q.notify <- struct{}{}:
	default:
	}
}
