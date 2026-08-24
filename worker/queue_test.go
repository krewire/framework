// Tests for KWF-L5H2F

package worker_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/krewire/framework/worker"
)

type jobFn func(context.Context) error

func (f jobFn) Run(ctx context.Context) error { return f(ctx) }

var errBoom = errors.New("boom")

func noopJob() jobFn {
	return jobFn(func(context.Context) error { return nil })
}

func TestFRK_SVC_060_EnqueueDequeueAckNack_Lifecycle(t *testing.T) {
	t.Run("PriorityOrderFIFOAmongEquals", func(t *testing.T) {
		q := worker.NewInMemoryQueue()
		ctx := context.Background()

		idP1, err := q.Enqueue(ctx, noopJob(), worker.Options{Priority: 1})
		mustNoErr(t, err)
		idP10A, err := q.Enqueue(ctx, noopJob(), worker.Options{Priority: 10})
		mustNoErr(t, err)
		idP5, err := q.Enqueue(ctx, noopJob(), worker.Options{Priority: 5})
		mustNoErr(t, err)
		idP10B, err := q.Enqueue(ctx, noopJob(), worker.Options{Priority: 10})
		mustNoErr(t, err)

		want := []worker.JobID{idP10A, idP10B, idP5, idP1}
		for _, wantID := range want {
			task := mustDequeue(t, q, time.Second)
			if task.ID != wantID {
				t.Fatalf("dequeue order: got %s, want %s", task.ID, wantID)
			}
			mustNoErr(t, q.Ack(ctx, task))
		}
	})

	t.Run("DelayGatesDelivery", func(t *testing.T) {
		q := worker.NewInMemoryQueue()
		ctx := context.Background()

		wantID, err := q.Enqueue(ctx, noopJob(), worker.Options{Delay: 40 * time.Millisecond})
		mustNoErr(t, err)

		gateCtx, cancel := context.WithTimeout(ctx, 5*time.Millisecond)
		defer cancel()
		if _, err := q.Dequeue(gateCtx); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("delayed task delivered early: err=%v", err)
		}

		task := mustDequeue(t, q, 2*time.Second)
		if task.ID != wantID {
			t.Fatalf("got %s, want %s", task.ID, wantID)
		}
	})

	t.Run("AckNackContract", func(t *testing.T) {
		q := worker.NewInMemoryQueue()
		ctx := context.Background()

		if _, err := q.Enqueue(ctx, nil, worker.Options{}); err == nil {
			t.Fatal("nil job accepted")
		}
		if err := q.Ack(ctx, nil); !errors.Is(err, worker.ErrUnknownTask) {
			t.Fatalf("Ack(nil): err=%v", err)
		}

		if _, err := q.Enqueue(ctx, noopJob(), worker.Options{}); err != nil {
			t.Fatalf("Enqueue: %v", err)
		}
		task := mustDequeue(t, q, time.Second)
		mustNoErr(t, q.Ack(ctx, task))
		if err := q.Ack(ctx, task); !errors.Is(err, worker.ErrUnknownTask) {
			t.Fatalf("double Ack: err=%v", err)
		}
		if err := q.Nack(ctx, task); !errors.Is(err, worker.ErrUnknownTask) {
			t.Fatalf("Nack after Ack: err=%v", err)
		}

		if _, err := q.Enqueue(ctx, noopJob(), worker.Options{
			Retry: &worker.RetryPolicy{MaxAttempts: 1},
		}); err != nil {
			t.Fatalf("Enqueue: %v", err)
		}
		task = mustDequeue(t, q, time.Second)
		task.LastErr = errBoom
		mustNoErr(t, q.Nack(ctx, task))
		dlq := q.DLQ()
		if len(dlq) != 1 || dlq[0].Err != errBoom || dlq[0].Attempts != 1 {
			t.Fatalf("DLQ = %+v", dlq)
		}
	})

	t.Run("CronRearmsAfterAck", func(t *testing.T) {
		q := worker.NewInMemoryQueue()
		ctx := context.Background()

		_, err := q.Enqueue(ctx, noopJob(), worker.Options{Cron: "* * * * *"})
		mustNoErr(t, err)

		task := mustDequeue(t, q, time.Second)
		mustNoErr(t, q.Ack(ctx, task))

		gateCtx, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
		defer cancel()
		if _, err := q.Dequeue(gateCtx); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("cron re-arm not gated until next fire: err=%v", err)
		}
	})
}

func TestFRK_SVC_062_RetryPolicy_DeadLettersAfterMaxAttempts(t *testing.T) {
	q := worker.NewInMemoryQueue()
	ctx := context.Background()

	policy := &worker.RetryPolicy{MaxAttempts: 3, InitialBackoff: time.Millisecond, MaxBackoff: 4 * time.Millisecond}
	wantID, err := q.Enqueue(ctx, jobFn(func(context.Context) error { return errBoom }), worker.Options{Retry: policy})
	mustNoErr(t, err)

	for attempt := 1; attempt <= 3; attempt++ {
		task := mustDequeue(t, q, time.Second)
		if task.ID != wantID {
			t.Fatalf("attempt %d: got %s, want %s", attempt, task.ID, wantID)
		}
		if task.Attempts != attempt {
			t.Fatalf("attempt %d: task.Attempts=%d", attempt, task.Attempts)
		}
		task.LastErr = errBoom
		if err := q.Nack(ctx, task); err != nil {
			t.Fatalf("Nack %d: %v", attempt, err)
		}
	}

	dlq := q.DLQ()
	if len(dlq) != 1 {
		t.Fatalf("len(DLQ)=%d, want 1", len(dlq))
	}
	dl := dlq[0]
	if dl.ID != wantID || dl.Err != errBoom || dl.Attempts != 3 || dl.At.IsZero() {
		t.Fatalf("dead letter = %+v", dl)
	}

	gateCtx, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
	defer cancel()
	if _, err := q.Dequeue(gateCtx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("exhausted task redelivered: err=%v", err)
	}
}

func TestFRK_SVC_062_RetryPolicy_BackoffTable(t *testing.T) {
	p := worker.RetryPolicy{MaxAttempts: 10, InitialBackoff: 10 * time.Millisecond, MaxBackoff: 50 * time.Millisecond}
	cases := []struct {
		n    int
		want time.Duration
	}{
		{0, 0},
		{1, 10 * time.Millisecond},
		{2, 20 * time.Millisecond},
		{3, 40 * time.Millisecond},
		{4, 50 * time.Millisecond},
		{9, 50 * time.Millisecond},
	}
	for _, c := range cases {
		if got := p.Backoff(c.n); got != c.want {
			t.Errorf("Backoff(%d) = %v, want %v", c.n, got, c.want)
		}
	}
	if got := (worker.RetryPolicy{}).Backoff(1); got != worker.DefaultRetryPolicy.InitialBackoff {
		t.Errorf("zero-value policy Backoff(1) = %v, want default initial %v",
			got, worker.DefaultRetryPolicy.InitialBackoff)
	}
}

func TestFRK_SVC_063_InMemoryBackend_NoExternalDeps_EndToEnd(t *testing.T) {
	q := worker.NewInMemoryQueue()
	ctx := context.Background()

	var ran atomic.Bool
	_, err := q.Enqueue(ctx, jobFn(func(context.Context) error { ran.Store(true); return nil }), worker.Options{})
	mustNoErr(t, err)

	poisonPolicy := &worker.RetryPolicy{MaxAttempts: 2, InitialBackoff: time.Millisecond, MaxBackoff: 2 * time.Millisecond}
	poisonID, err := q.Enqueue(ctx, jobFn(func(context.Context) error { return errBoom }),
		worker.Options{Priority: -1, Retry: poisonPolicy})
	mustNoErr(t, err)

	r := worker.NewRunner(q, worker.WithConcurrency(4), worker.WithLogger(discardLogger()))
	done := make(chan error, 1)
	go func() { done <- r.Run(ctx) }()

	waitFor(t, func() bool { return ran.Load() && len(q.DLQ()) == 1 })
	r.Stop()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after Stop")
	}

	if !ran.Load() {
		t.Fatal("success job never ran")
	}
	dlq := q.DLQ()
	if len(dlq) != 1 || dlq[0].ID != poisonID || dlq[0].Attempts != 2 || dlq[0].Err != errBoom {
		t.Fatalf("DLQ = %+v", dlq)
	}

	snapshot := q.DLQ()
	snapshot[0].ID = "mutated"
	if q.DLQ()[0].ID == "mutated" {
		t.Fatal("DLQ snapshot is not a copy")
	}
}

func mustNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func mustDequeue(t *testing.T, q worker.Queue, timeout time.Duration) *worker.Task {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	task, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Dequeue: %v", err)
	}
	return task
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition not met within 2s")
}
