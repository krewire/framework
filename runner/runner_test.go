package runner_test

import (
	"context"
	"testing"

	"github.com/krewire/framework/runner"
	"github.com/krewire/framework/service"
)

type dummyRegistry struct{}

func (dummyRegistry) Set(string, any) error  { return nil }
func (dummyRegistry) Get(string) (any, bool) { return nil, false }

func TestFunc_Run(t *testing.T) {
	called := false
	r := runner.Func(func(_ context.Context, _ service.Registry) error {
		called = true
		return nil
	})

	if err := r.Run(context.Background(), dummyRegistry{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected runner func to be called")
	}

	// Nil func does not error
	var nilFunc runner.Func
	if err := nilFunc.Run(context.Background(), dummyRegistry{}); err != nil {
		t.Fatalf("nil func should return nil: %v", err)
	}
}
