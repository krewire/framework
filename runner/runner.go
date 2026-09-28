// Package runner defines the runtime contract for Krewire workloads.
package runner

import (
	"context"

	"github.com/krewire/framework/service"
)

// Runner executes one application runtime after services have been registered
// and started. Implementations may represent an HTTP server, CLI, worker,
// scheduler, event consumer, or another workload runtime.
type Runner interface {
	Run(context.Context, service.Registry) error
}

// Func adapts a function into a Runner.
type Func func(context.Context, service.Registry) error

// Run implements Runner.
func (f Func) Run(ctx context.Context, registry service.Registry) error {
	if f == nil {
		return nil
	}
	return f(ctx, registry)
}

var _ Runner = Func(nil)
