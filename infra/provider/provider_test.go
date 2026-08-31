// Tests for KWF-B7N3D
package provider

import (
	"context"
	"testing"
)

// fakeProvider is a test double for Provider.
type fakeProvider struct {
	name      string
	resources []Resource
	created   []Resource
	deleted   []string
}

func (f *fakeProvider) Name() string { return f.name }

func (f *fakeProvider) Create(ctx context.Context, r Resource) error {
	f.created = append(f.created, r)
	f.resources = append(f.resources, r)
	return nil
}

func (f *fakeProvider) Read(ctx context.Context, kind, id string) (Resource, error) {
	for _, r := range f.resources {
		if r.Kind == kind && r.ID == id {
			return r, nil
		}
	}
	return Resource{}, ErrNotFound{Kind: kind, ID: id}
}

func (f *fakeProvider) Update(ctx context.Context, r Resource) error {
	for i, existing := range f.resources {
		if existing.Kind == r.Kind && existing.ID == r.ID {
			f.resources[i] = r
			return nil
		}
	}
	return ErrNotFound{Kind: r.Kind, ID: r.ID}
}

func (f *fakeProvider) Delete(ctx context.Context, kind, id string) error {
	f.deleted = append(f.deleted, kind+":"+id)
	return nil
}

func (f *fakeProvider) Plan(ctx context.Context, desired []Resource) (Plan, error) {
	plan := Plan{}
	existing := map[string]Resource{}
	for _, r := range f.resources {
		existing[r.Kind+"/"+r.ID] = r
	}
	for _, d := range desired {
		key := d.Kind + "/" + d.ID
		if _, ok := existing[key]; !ok {
			plan.Actions = append(plan.Actions, Action{Op: OpCreate, Resource: d, Reason: "new resource"})
		}
	}
	for key, r := range existing {
		found := false
		for _, d := range desired {
			if d.Kind+"/"+d.ID == key {
				found = true
				break
			}
		}
		if !found {
			plan.Actions = append(plan.Actions, Action{Op: OpDelete, Resource: r, Reason: "removed from desired"})
		}
	}
	return plan, nil
}

func (f *fakeProvider) Resources() []ResourceSchema {
	return []ResourceSchema{{Kind: "Compute", Description: "compute resource"}}
}

// Spec: KWF-B7N3D FRK-INFRA-001 Scope: Unit
func TestProvider_CreateAndRead(t *testing.T) {
	p := &fakeProvider{name: "test"}
	ctx := context.Background()
	r := Resource{Kind: "Compute", ID: "web-1", Properties: map[string]any{"image": "nginx"}}
	if err := p.Create(ctx, r); err != nil {
		t.Fatal(err)
	}
	got, err := p.Read(ctx, "Compute", "web-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "web-1" {
		t.Fatalf("got ID %q", got.ID)
	}
}

// Spec: KWF-B7N3D FRK-INFRA-001 Scope: Unit
func TestProvider_ReadNotFound(t *testing.T) {
	p := &fakeProvider{name: "test"}
	_, err := p.Read(context.Background(), "Compute", "missing")
	if !IsNotFound(err) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// Spec: KWF-B7N3D FRK-INFRA-002 Scope: Unit
func TestPlan_HasChanges(t *testing.T) {
	plan := Plan{Actions: []Action{{Op: OpCreate}}}
	if !plan.HasChanges() {
		t.Fatal("expected changes")
	}
	plan = Plan{Actions: []Action{{Op: OpNoOp}}}
	if plan.HasChanges() {
		t.Fatal("expected no changes")
	}
}

// Spec: KWF-B7N3D FRK-INFRA-003 Scope: Unit
func TestResourceSchema_ValidateResource(t *testing.T) {
	called := false
	s := ResourceSchema{
		Kind: "Test",
		Validate: func(props map[string]any) error {
			called = true
			return nil
		},
	}
	if err := s.ValidateResource(Resource{Kind: "Test"}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("validator not called")
	}
}

// Spec: KWF-B7N3D FRK-INFRA-003 Scope: Unit
func TestResourceSchema_ValidateResource_NilValidator(t *testing.T) {
	s := ResourceSchema{Kind: "Test"}
	if err := s.ValidateResource(Resource{Kind: "Test"}); err != nil {
		t.Fatalf("nil validator should pass: %v", err)
	}
}
