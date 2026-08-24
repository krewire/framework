package worker

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
)

// Runner drains a Queue: it dequeues tasks with bounded concurrency, invokes
// Job.Run, Acks successes, and Nacks failures so the queue applies retries
// and dead-lettering (FRK-SVC-062). Run is not safe for concurrent calls;
// Stop may be called from any goroutine.
type Runner struct {
	queue       Queue
	log         *slog.Logger
	concurrency int

	mu   sync.Mutex
	stop chan struct{}
}

// RunnerOption configures a Runner.
type RunnerOption func(*Runner)

// WithConcurrency bounds how many jobs run at once (default GOMAXPROCS).
func WithConcurrency(n int) RunnerOption {
	return func(r *Runner) {
		if n > 0 {
			r.concurrency = n
		}
	}
}

// WithLogger injects a structured logger; the default is slog.Default()
// (KWF-L5H2F NFR3).
func WithLogger(l *slog.Logger) RunnerOption {
	return func(r *Runner) {
		if l != nil {
			r.log = l
		}
	}
}

// NewRunner returns a Runner over queue.
func NewRunner(q Queue, opts ...RunnerOption) *Runner {
	r := &Runner{
		queue:       q,
		log:         slog.Default(),
		concurrency: runtime.NumCPU(),
		stop:        make(chan struct{}),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Stop asks Run to stop dequeuing and drain in-flight jobs. Jobs observe
// cancellation only through the context passed to Run.
func (r *Runner) Stop() {
	r.mu.Lock()
	st := r.stop
	r.mu.Unlock()
	if st != nil {
		select {
		case <-st:
		default:
			close(st)
		}
	}
}

// Run blocks until Stop is called or ctx is cancelled, then waits for all
// in-flight jobs. It returns nil after a graceful Stop and ctx.Err() when ctx
// was cancelled.
func (r *Runner) Run(ctx context.Context) error {
	r.mu.Lock()
	r.stop = make(chan struct{})
	stop := r.stop
	r.mu.Unlock()

	dequeueCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		select {
		case <-ctx.Done():
		case <-stop:
		}
		cancel()
	}()

	var wg sync.WaitGroup
	for i := 0; i < r.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				task, err := r.queue.Dequeue(dequeueCtx)
				if err != nil {
					return
				}
				r.execute(ctx, task)
			}
		}()
	}
	<-dequeueCtx.Done()
	wg.Wait()
	return ctx.Err()
}

func (r *Runner) execute(ctx context.Context, task *Task) {
	err := r.invoke(ctx, task)
	if err == nil {
		r.log.Debug("worker: job done", slog.String("job_id", string(task.ID)), slog.Int("attempt", task.Attempts))
		if ackErr := r.queue.Ack(ctx, task); ackErr != nil {
			r.log.Warn("worker: ack failed", slog.String("job_id", string(task.ID)), slog.Any("error", ackErr))
		}
		return
	}
	task.LastErr = err
	r.log.Warn("worker: job failed",
		slog.String("job_id", string(task.ID)),
		slog.Int("attempt", task.Attempts),
		slog.Any("error", err))
	if nackErr := r.queue.Nack(ctx, task); nackErr != nil {
		r.log.Error("worker: nack failed", slog.String("job_id", string(task.ID)), slog.Any("error", nackErr))
	}
}

func (r *Runner) invoke(ctx context.Context, task *Task) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("worker: job %s panicked: %v", task.ID, p)
		}
	}()
	return task.Job.Run(ctx)
}
