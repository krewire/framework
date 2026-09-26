package worker_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/krewire/framework/storage"
	"github.com/krewire/framework/worker"
)

type mockEmailJob struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
}

func (m *mockEmailJob) JobType() string          { return "email" }
func (m *mockEmailJob) Encode() ([]byte, error) { return json.Marshal(m) }
func (m *mockEmailJob) Decode(b []byte) error   { return json.Unmarshal(b, m) }
func (m *mockEmailJob) Run(ctx context.Context) error {
	return nil
}

func TestKVQueue_BasicFlow(t *testing.T) {
	ctx := context.Background()
	kv := storage.NewMemory()

	q, err := worker.NewKVQueue(ctx, kv)
	if err != nil {
		t.Fatalf("failed to create KVQueue: %v", err)
	}

	job := &mockEmailJob{To: "user@example.com", Subject: "Welcome"}
	id, err := q.Enqueue(ctx, job, worker.Options{Priority: 10})
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty job ID")
	}

	task, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("dequeue failed: %v", err)
	}
	if task.ID != id {
		t.Fatalf("expected job ID %s, got %s", id, task.ID)
	}

	if err := q.Ack(ctx, task); err != nil {
		t.Fatalf("ack failed: %v", err)
	}
}

func TestKVQueue_PersistenceAcrossRestarts(t *testing.T) {
	ctx := context.Background()
	kv := storage.NewMemory()

	reg := worker.NewJobRegistry()
	reg.Register("email", func() worker.SerializableJob { return &mockEmailJob{} })

	// Process 1: Enqueue job and shutdown
	q1, err := worker.NewKVQueue(ctx, kv, worker.WithJobRegistry(reg))
	if err != nil {
		t.Fatalf("failed to create q1: %v", err)
	}

	job := &mockEmailJob{To: "persist@test.com", Subject: "Important"}
	id, err := q1.Enqueue(ctx, job, worker.Options{Priority: 5})
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	// Process 2: Start new queue from the same KV store
	q2, err := worker.NewKVQueue(ctx, kv, worker.WithJobRegistry(reg))
	if err != nil {
		t.Fatalf("failed to restore q2: %v", err)
	}

	task, err := q2.Dequeue(ctx)
	if err != nil {
		t.Fatalf("dequeue from restored queue failed: %v", err)
	}
	if task.ID != id {
		t.Fatalf("expected restored job ID %s, got %s", id, task.ID)
	}

	emailJob, ok := task.Job.(*mockEmailJob)
	if !ok {
		t.Fatalf("expected job to be decoded as *mockEmailJob, got %T", task.Job)
	}
	if emailJob.To != "persist@test.com" {
		t.Fatalf("expected recipient persist@test.com, got %s", emailJob.To)
	}

	if err := q2.Ack(ctx, task); err != nil {
		t.Fatalf("ack failed: %v", err)
	}
}

func TestKVQueue_RetryAndDLQ(t *testing.T) {
	ctx := context.Background()
	kv := storage.NewMemory()

	q, err := worker.NewKVQueue(ctx, kv)
	if err != nil {
		t.Fatalf("failed to create KVQueue: %v", err)
	}

	job := &mockEmailJob{To: "fail@test.com", Subject: "Test"}
	id, err := q.Enqueue(ctx, job, worker.Options{
		Retry: &worker.RetryPolicy{
			MaxAttempts:    2,
			InitialBackoff: time.Millisecond,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// 1st Attempt -> Nack
	task1, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatal(err)
	}
	task1.LastErr = errors.New("network down")
	if err := q.Nack(ctx, task1); err != nil {
		t.Fatal(err)
	}

	// 2nd Attempt -> Nack -> Exceeded MaxAttempts -> DLQ
	time.Sleep(5 * time.Millisecond)
	task2, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatal(err)
	}
	task2.LastErr = errors.New("permanent failure")
	if err := q.Nack(ctx, task2); err != nil {
		t.Fatal(err)
	}

	dlq := q.DLQ()
	if len(dlq) != 1 {
		t.Fatalf("expected 1 dead letter, got %d", len(dlq))
	}
	if dlq[0].ID != id {
		t.Fatalf("expected DLQ job ID %s, got %s", id, dlq[0].ID)
	}
}

func TestKVQueue_RunnerIntegration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	kv := storage.NewMemory()
	q, err := worker.NewKVQueue(ctx, kv)
	if err != nil {
		t.Fatal(err)
	}

	var executed atomic.Int32
	job := &worker.PayloadJob{
		Type: "counter",
		Handler: func(ctx context.Context, payload []byte) error {
			executed.Add(1)
			return nil
		},
	}

	for i := 0; i < 5; i++ {
		if _, err := q.Enqueue(ctx, job, worker.Options{}); err != nil {
			t.Fatal(err)
		}
	}

	runner := worker.NewRunner(q, worker.WithConcurrency(2))
	go func() {
		_ = runner.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	runner.Stop()

	if executed.Load() != 5 {
		t.Fatalf("expected 5 jobs executed by runner, got %d", executed.Load())
	}
}
