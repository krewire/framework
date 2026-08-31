// Package provider defines the multi-cloud infrastructure contract (KWF-B7N3D
// FRK-INFRA-001/002/003/004). A Provider implements CRUD and plan/apply for a
// set of resource kinds. Resources carry dependencies; plans are topologically
// sorted. All operations are side-effect free until Apply.
package provider

import (
	"context"
	"fmt"
)

// Op is the action a plan step takes (FRK-INFRA-002).
type Op string

const (
	OpCreate Op = "create"
	OpUpdate Op = "update"
	OpDelete Op = "delete"
	OpNoOp   Op = "noop"
)

// Resource is a declarative infrastructure primitive (FRK-INFRA-004).
type Resource struct {
	Kind       string            `json:"kind" validate:"required"`
	ID         string            `json:"id" validate:"required"`
	Properties map[string]any    `json:"properties,omitempty"`
	DependsOn  []string          `json:"dependsOn,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// Action is one step of a plan (FRK-INFRA-002).
type Action struct {
	Op       Op       `json:"op"`
	Resource Resource `json:"resource"`
	Reason   string   `json:"reason,omitempty"`
}

// Plan is an ordered, dependency-aware set of actions (FRK-INFRA-002).
type Plan struct {
	Actions []Action `json:"actions"`
}

// HasChanges reports whether the plan mutates infrastructure.
func (p Plan) HasChanges() bool {
	for _, a := range p.Actions {
		if a.Op != OpNoOp {
			return true
		}
	}
	return false
}

// Provider is the multi-cloud infrastructure contract (FRK-INFRA-001).
type Provider interface {
	Name() string
	Create(ctx context.Context, r Resource) error
	Read(ctx context.Context, kind, id string) (Resource, error)
	Update(ctx context.Context, r Resource) error
	Delete(ctx context.Context, kind, id string) error
	Plan(ctx context.Context, desired []Resource) (Plan, error)
	Resources() []ResourceSchema
}

// ResourceSchema describes a kind a provider supports (FRK-INFRA-003).
type ResourceSchema struct {
	Kind        string `json:"kind"`
	Description string `json:"description"`
	Validate    func(props map[string]any) error
}

// Validate checks a resource against its provider schema (FRK-INFRA-010).
func (s ResourceSchema) ValidateResource(r Resource) error {
	if s.Validate == nil {
		return nil
	}
	return s.Validate(r.Properties)
}

// ErrNotFound signals a resource does not exist (FRK-INFRA-001).
type ErrNotFound struct {
	Kind string
	ID   string
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("%s %q not found", e.Kind, e.ID)
}

// IsNotFound reports whether err is ErrNotFound.
func IsNotFound(err error) bool {
	_, ok := err.(ErrNotFound)
	return ok
}
