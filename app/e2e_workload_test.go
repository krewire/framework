package app_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/krewire/framework/app"
	"github.com/krewire/framework/web"
	"github.com/krewire/framework/worker"
)

type dummyJob struct {
	done chan struct{}
}

func (j *dummyJob) Run(ctx context.Context) error {
	close(j.done)
	return nil
}

func TestApplication_RunsWebAndWorkerConcurrently(t *testing.T) {
	appl := app.NewApplication()

	// 1. Setup Web App
	webApp := web.NewApp()
	webApp.Router().Get("/ping", func(w http.ResponseWriter, req *http.Request, _ web.Params) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	})

	// 2. Setup Worker
	q := worker.NewInMemoryQueue()
	jobDone := make(chan struct{})
	_, err := q.Enqueue(context.Background(), &dummyJob{done: jobDone}, worker.Options{})
	if err != nil {
		t.Fatal(err)
	}
	workerRunner := worker.NewRunner(q)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	runErrCh := make(chan error, 1)
	go func() {
		// Run both workloads under one application lifecycle
		runErrCh <- appl.Run(ctx, webApp.Runner("127.0.0.1:0"), workerRunner.AsRunner())
	}()

	// Wait for worker job to complete
	select {
	case <-jobDone:
		// worker executed successfully
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for worker job")
	}

	cancel() // Trigger graceful stop of all workloads
	if err := <-runErrCh; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("unexpected run error: %v", err)
	}
}
