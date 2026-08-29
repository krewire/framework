package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type startStopProvider struct {
	starts []string
	stops  []string
	name   string
	fail   error
}

func (p *startStopProvider) Register(c *Container) error { return nil }
func (p *startStopProvider) Boot(c *Container) error     { return nil }
func (p *startStopProvider) Start(ctx context.Context) error {
	if p.fail != nil {
		return p.fail
	}
	p.starts = append(p.starts, p.name)
	return nil
}
func (p *startStopProvider) Stop(ctx context.Context) error {
	p.stops = append(p.stops, p.name)
	return nil
}

func TestLifecycleHooksOrder(t *testing.T) {
	a := NewApp()
	var order []string
	a.OnStart(func(ctx context.Context) error { order = append(order, "hook1-start"); return nil })
	a.AddHook(Hook{OnStart: func(ctx context.Context) error { order = append(order, "hook2-start"); return nil }})
	a.OnStop(func(ctx context.Context) error { order = append(order, "hook1-stop"); return nil })
	a.AddHook(Hook{OnStop: func(ctx context.Context) error { order = append(order, "hook2-stop"); return nil }})

	c, err := a.Build()
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Start(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 || order[0] != "hook1-start" || order[1] != "hook2-start" {
		t.Fatalf("start order = %v want [hook1-start hook2-start]", order)
	}
	if err := a.Stop(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	// Stop reverse: hook2-stop then hook1-stop, plus earlier starts remain
	if len(order) != 4 || order[2] != "hook2-stop" || order[3] != "hook1-stop" {
		t.Fatalf("stop order = %v want [hook1-start hook2-start hook2-stop hook1-stop]", order)
	}
}

func TestProviderStarterStopperOrder(t *testing.T) {
	p1 := &startStopProvider{name: "p1"}
	p2 := &startStopProvider{name: "p2"}
	a := NewApp(p1, p2)
	c, err := a.Build()
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Start(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	if len(p1.starts) != 1 || len(p2.starts) != 1 {
		t.Fatalf("starts p1=%v p2=%v", p1.starts, p2.starts)
	}
	// providers start in registration order: p1 then p2
	// hooks would be after; we just check providers called
	if err := a.Stop(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	// stop reverse: p2 then p1
	if len(p1.stops) != 1 || len(p2.stops) != 1 {
		t.Fatalf("stops p1=%v p2=%v", p1.stops, p2.stops)
	}
}

func TestStartFailsFast(t *testing.T) {
	fail := errors.New("start boom")
	p1 := &startStopProvider{name: "p1"}
	p2 := &startStopProvider{name: "p2", fail: fail}
	a := NewApp(p1, p2)
	a.OnStart(func(ctx context.Context) error { t.Fatal("hook should not run after provider failure"); return nil })
	c, err := a.Build()
	if err != nil {
		t.Fatal(err)
	}
	err = a.Start(context.Background(), c)
	if !errors.Is(err, fail) {
		t.Fatalf("expected fail, got %v", err)
	}
}

func TestHookStopError(t *testing.T) {
	a := NewApp()
	a.OnStop(func(ctx context.Context) error { return errors.New("stop fail") })
	a.OnStop(func(ctx context.Context) error { return nil })
	c, err := a.Build()
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Start(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	err = a.Stop(context.Background(), c)
	if err == nil || !strings.Contains(err.Error(), "stop fail") {
		t.Fatalf("expected stop fail, got %v", err)
	}
}

func TestBootstrapStartsAndStops(t *testing.T) {
	p := &startStopProvider{name: "p1"}
	ctx := context.Background()
	app, err := Bootstrap(ctx, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.starts) != 1 {
		t.Fatalf("bootstrap should start, got %v", p.starts)
	}
	if app.Container() == nil {
		t.Fatal("container nil")
	}
	if err := app.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	if len(p.stops) != 1 {
		t.Fatalf("stop should be called, got %v", p.stops)
	}
}

func TestBootstrapWithOptions(t *testing.T) {
	p := &startStopProvider{name: "p1"}
	ctx := context.Background()
	app, err := BootstrapWithOptions(ctx, []Option{WithTrace()}, p)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Stop(ctx)
	if app.Container() == nil {
		t.Fatal("container nil")
	}
}

func TestApplicationRunGraceful(t *testing.T) {
	p := &startStopProvider{name: "p1"}
	ctx, cancel := context.WithCancel(context.Background())
	app, err := Bootstrap(ctx, p)
	if err != nil {
		t.Fatal(err)
	}
	// Run waits for ctx cancellation
	done := make(chan error, 1)
	go func() { done <- app.Run(ctx) }()
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
	if len(p.stops) != 1 {
		t.Fatalf("Run should stop on cancel, stops=%v", p.stops)
	}
}
