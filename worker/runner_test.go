// Tests for KWF-L5H2F

package worker_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/krewire/framework/worker"
)

func TestFRK_SVC_062_Runner_AcksSuccessAndRetriesFailure(t *testing.T) {
	q := worker.NewInMemoryQueue()
	ctx := context.Background()

	var flakyRuns atomic.Int32
	flakyPolicy := &worker.RetryPolicy{MaxAttempts: 5, InitialBackoff: time.Millisecond, MaxBackoff: 4 * time.Millisecond}
	_, err := q.Enqueue(ctx, jobFn(func(context.Context) error {
		if flakyRuns.Add(1) <= 2 {
			return errBoom
		}
		return nil
	}), worker.Options{Retry: flakyPolicy})
	mustNoErr(t, err)

	poisonPolicy := &worker.RetryPolicy{MaxAttempts: 2, InitialBackoff: time.Millisecond, MaxBackoff: 2 * time.Millisecond}
	_, err = q.Enqueue(ctx, jobFn(func(context.Context) error { return errBoom }), worker.Options{Retry: poisonPolicy})
	mustNoErr(t, err)

	var successRuns atomic.Int32
	_, err = q.Enqueue(ctx, jobFn(func(context.Context) error { successRuns.Add(1); return nil }), worker.Options{})
	mustNoErr(t, err)

	r := worker.NewRunner(q,
		worker.WithConcurrency(3),
		worker.WithLogger(discardLogger()))
	done := make(chan error, 1)
	go func() { done <- r.Run(ctx) }()

	waitFor(t, func() bool {
		return flakyRuns.Load() >= 3 && successRuns.Load() >= 1 && len(q.DLQ()) == 1
	})
	r.Stop()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after Stop")
	}

	if got := flakyRuns.Load(); got != 3 {
		t.Errorf("flaky runs after drain = %d, want exactly 3", got)
	}
	if got := successRuns.Load(); got != 1 {
		t.Errorf("success runs after drain = %d, want exactly 1 (acked)", got)
	}
	dlq := q.DLQ()
	if len(dlq) != 1 || dlq[0].Attempts != 2 || !errors.Is(dlq[0].Err, errBoom) {
		t.Errorf("DLQ = %+v", dlq)
	}
}

func TestFRK_SVC_060_Runner_ContextCancellationReachesJobs(t *testing.T) {
	q := worker.NewInMemoryQueue()
	ctx, cancel := context.WithCancel(context.Background())

	started := make(chan struct{}, 8)
	sawCancel := make(chan struct{}, 8)
	_, err := q.Enqueue(ctx, jobFn(func(jobCtx context.Context) error {
		started <- struct{}{}
		<-jobCtx.Done()
		sawCancel <- struct{}{}
		return jobCtx.Err()
	}), worker.Options{})
	mustNoErr(t, err)

	r := worker.NewRunner(q, worker.WithConcurrency(1), worker.WithLogger(discardLogger()))
	done := make(chan error, 1)
	go func() { done <- r.Run(ctx) }()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("job never started")
	}
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Run = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
	select {
	case <-sawCancel:
	default:
		t.Fatal("job context was never cancelled")
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
