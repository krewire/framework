// Tests for KWF-B7N3D
package schema

import (
	"testing"
)

// Spec: KWF-B7N3D FRK-INFRA-010 Scope: Unit
func TestValidate_Compute_Valid(t *testing.T) {
	c := Compute{Image: "nginx:latest", Runtime: "fargate", Replicas: 2}
	if err := Validate(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Spec: KWF-B7N3D FRK-INFRA-010 Scope: Unit
func TestValidate_Compute_Invalid(t *testing.T) {
	c := Compute{Runtime: "fargate"} // missing required image
	if err := Validate(c); err == nil {
		t.Fatal("expected validation error for missing image")
	}
}

// Spec: KWF-B7N3D FRK-INFRA-010 Scope: Unit
func TestValidate_Database_Valid(t *testing.T) {
	d := Database{Engine: "postgres", Storage: 100, SecretID: "db-creds"}
	if err := Validate(d); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Spec: KWF-B7N3D FRK-INFRA-010 Scope: Unit
func TestValidate_SecretRef_Valid(t *testing.T) {
	s := SecretRef{Key: "API_KEY", Env: "API_KEY"}
	if err := Validate(s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
