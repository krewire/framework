package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/krewire/framework/app"
	"github.com/krewire/framework/service"
)

type dummyProvider struct {
	name    string
	started bool
	stopped bool
}

func (p *dummyProvider) Name() string { return p.name }
func (p *dummyProvider) Register(r service.Registry) error {
	return r.Set(p.name, p)
}
func (p *dummyProvider) Start(ctx context.Context, r service.Registry) error {
	p.started = true
	return nil
}
func (p *dummyProvider) Stop(ctx context.Context, r service.Registry) error {
	p.stopped = true
	return nil
}

type failProvider struct {
	dummyProvider
	failStart bool
}

func (p *failProvider) Start(ctx context.Context, r service.Registry) error {
	if p.failStart {
		return errors.New("start failed")
	}
	return nil
}

func TestApplication_UseAndLifecycle(t *testing.T) {
	appl := app.NewApplication()
	p1 := &dummyProvider{name: "p1"}
	p2 := &dummyProvider{name: "p2"}

	if err := appl.Use(p1, p2); err != nil {
		t.Fatalf("use failed: %v", err)
	}

	// Duplicate provider rejected
	if err := appl.Use(p1); err == nil {
		t.Fatal("expected error on duplicate provider")
	}

	ctx := context.Background()
	if err := appl.Bootstrap(ctx); err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}

	if !p1.started || !p2.started {
		t.Fatalf("expected all providers started, got p1=%v p2=%v", p1.started, p2.started)
	}

	// Verify services registered in container
	c := appl.Container()
	val, ok := c.Get("p1")
	if !ok || val != p1 {
		t.Fatalf("expected p1 in container, got %v", val)
	}

	typed, ok := app.ResolveNamed[*dummyProvider](c, "p2")
	if !ok || typed != p2 {
		t.Fatalf("expected typed p2, got %v", typed)
	}

	if err := appl.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}

	if !p1.stopped || !p2.stopped {
		t.Fatalf("expected all providers stopped, got p1=%v p2=%v", p1.stopped, p2.stopped)
	}
}

func TestApplication_RunWithRunner(t *testing.T) {
	appl := app.NewApplication()
	p := &dummyProvider{name: "db"}
	if err := appl.Use(p); err != nil {
		t.Fatal(err)
	}

	runnerCalled := false
	r := app.RunnerFunc(func(ctx context.Context, reg service.Registry) error {
		runnerCalled = true
		val, ok := reg.Get("db")
		if !ok || val != p {
			t.Errorf("expected db in registry")
		}
		return nil
	})

	if err := appl.Run(context.Background(), r); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if !runnerCalled {
		t.Fatal("expected runner to be executed")
	}

	// Providers should be stopped after Run
	if !p.stopped {
		t.Fatal("expected provider to be stopped after run completes")
	}
}

func TestApplication_StartFailureRollsBack(t *testing.T) {
	appl := app.NewApplication()
	p1 := &dummyProvider{name: "p1"}
	p2 := &failProvider{dummyProvider: dummyProvider{name: "p2"}, failStart: true}

	if err := appl.Use(p1, p2); err != nil {
		t.Fatal(err)
	}

	err := appl.Bootstrap(context.Background())
	if err == nil {
		t.Fatal("expected bootstrap error")
	}

	// p1 was started before p2 failed, so p1 should have been stopped in rollback
	if !p1.stopped {
		t.Fatal("expected p1 to be stopped after p2 start failure")
	}
}

func TestContainer_NamedServices(t *testing.T) {
	c := app.NewContainer()

	if err := c.Set("", "val"); err == nil {
		t.Fatal("expected error on empty name")
	}
	if err := c.Set("k", nil); err == nil {
		t.Fatal("expected error on nil value")
	}

	if err := c.Set("msg", "hello"); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	// Duplicate rejected
	if err := c.Set("msg", "world"); err == nil {
		t.Fatal("expected duplicate set to fail")
	}

	msg, ok := c.Get("msg")
	if !ok || msg != "hello" {
		t.Fatalf("expected hello, got %v", msg)
	}

	typed, ok := app.ResolveNamed[string](c, "msg")
	if !ok || typed != "hello" {
		t.Fatalf("expected hello, got %v", typed)
	}

	_, ok = app.ResolveNamed[int](c, "msg")
	if ok {
		t.Fatal("expected type mismatch to return false")
	}

	_, ok = app.ResolveNamed[string](c, "nonexistent")
	if ok {
		t.Fatal("expected nonexistent key to return false")
	}
}
