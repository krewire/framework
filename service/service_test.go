package service_test

import (
	"context"
	"testing"

	"github.com/krewire/framework/service"
)

type mockRegistry struct {
	data map[string]any
}

func (m *mockRegistry) Set(name string, value any) error {
	m.data[name] = value
	return nil
}

func (m *mockRegistry) Get(name string) (any, bool) {
	val, ok := m.data[name]
	return val, ok
}

type mockProvider struct {
	name    string
	started bool
	stopped bool
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) Register(r service.Registry) error {
	return r.Set(m.name, "registered")
}
func (m *mockProvider) Start(ctx context.Context, r service.Registry) error {
	m.started = true
	return nil
}
func (m *mockProvider) Stop(ctx context.Context, r service.Registry) error {
	m.stopped = true
	return nil
}

func TestServiceContracts(t *testing.T) {
	reg := &mockRegistry{data: make(map[string]any)}
	p := &mockProvider{name: "test"}

	var _ service.Registry = reg
	var _ service.Provider = p
	var _ service.Starter = p
	var _ service.Stopper = p

	if err := p.Register(reg); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	val, ok := reg.Get("test")
	if !ok || val != "registered" {
		t.Fatalf("expected registered, got %v", val)
	}

	ctx := context.Background()
	if err := p.Start(ctx, reg); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if !p.started {
		t.Fatal("expected provider to be started")
	}

	if err := p.Stop(ctx, reg); err != nil {
		t.Fatalf("stop failed: %v", err)
	}
	if !p.stopped {
		t.Fatal("expected provider to be stopped")
	}
}
