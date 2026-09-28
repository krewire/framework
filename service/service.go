// Package service defines contracts for application service providers.
package service

import "context"

// Registry is the dependency boundary exposed to providers during registration
// and lifecycle execution.
type Registry interface {
	Set(name string, value any) error
	Get(name string) (any, bool)
}

// Provider contributes one named service to an application.
type Provider interface {
	Name() string
	Register(Registry) error
}

// Starter is optionally implemented by providers that need startup work after
// all providers have registered their services.
type Starter interface {
	Start(context.Context, Registry) error
}

// Stopper is optionally implemented by providers that need graceful cleanup.
type Stopper interface {
	Stop(context.Context, Registry) error
}
