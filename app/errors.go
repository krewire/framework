package app

import (
	"fmt"
	"strings"
)

// ResolveError reports a failure building a dependency, carrying the path of
// types involved.
type ResolveError struct {
	// Type is the type that failed to resolve.
	Type string
	// Path is the chain of types from the requested root to Type.
	Path []string
	// Err is the underlying cause.
	Err error
}

// Error implements the error interface.
func (e *ResolveError) Error() string {
	if len(e.Path) > 0 {
		return fmt.Sprintf("app: resolve %s: %v (path: %s)", e.Type, e.Err, strings.Join(e.Path, " -> "))
	}
	return fmt.Sprintf("app: resolve %s: %v", e.Type, e.Err)
}

// Unwrap exposes the underlying cause for errors.Is/As.
func (e *ResolveError) Unwrap() error { return e.Err }

// Is supports errors.Is matching on the failing type.
func (e *ResolveError) Is(target error) bool {
	other, ok := target.(*ResolveError)
	return ok && other.Type == e.Type
}

// CycleError reports a circular dependency.
type CycleError struct {
	// Path is the cycle, starting and ending at the same type.
	Path []string
}

// Error implements the error interface.
func (e *CycleError) Error() string {
	return "app: circular dependency: " + strings.Join(e.Path, " -> ")
}

// Errors returned when a binding is absent or the container is locked.
var (
	// ErrLocked is returned by Provide/Singleton/Override after the first
	// resolution (FRK-CNT-009).
	ErrLocked = fmt.Errorf("app: container locked after first resolution")
)
