// Package schema provides typed resource schemas and validation for common
// infrastructure primitives (KWF-B7N3D FRK-INFRA-010/011/012). Schemas map Go
// structs with validation tags to JSON schema via libs/validate.
package schema

import (
	"fmt"

	"github.com/krewire/libs/validate"
)

// Kind constants for common infrastructure primitives (FRK-INFRA-011).
const (
	KindCompute     = "Compute"
	KindDatabase    = "Database"
	KindStorage     = "Storage"
	KindNetwork     = "Network"
	KindDNS         = "DNS"
	KindCertificate = "Certificate"
	KindSecretRef   = "SecretRef"
)

// Compute describes a compute resource (FRK-INFRA-011).
type Compute struct {
	Image    string            `json:"image" validate:"required"`
	Runtime  string            `json:"runtime" validate:"oneof=ecs fargate lambda"`
	Env      map[string]string `json:"env,omitempty"`
	Replicas int               `json:"replicas" validate:"min=1,max=100"`
}

// Database describes a database resource (FRK-INFRA-011).
type Database struct {
	Engine   string `json:"engine" validate:"required,oneof=postgres mysql dynamodb"`
	Version  string `json:"version,omitempty"`
	Storage  int    `json:"storage" validate:"min=20"`
	SecretID string `json:"secretId" validate:"required"`
}

// Storage describes a storage resource (FRK-INFRA-011).
type Storage struct {
	Bucket   string `json:"bucket" validate:"required"`
	Transfer bool   `json:"transfer,omitempty"`
}

// Network describes a network resource (FRK-INFRA-011).
type Network struct {
	CIDR    string   `json:"cidr" validate:"required"`
	Subnets []string `json:"subnets,omitempty"`
}

// DNS describes a DNS resource (FRK-INFRA-011).
type DNS struct {
	Domain string `json:"domain" validate:"required"`
	Target string `json:"target" validate:"required"`
}

// Certificate describes a certificate resource (FRK-INFRA-011).
type Certificate struct {
	Domain string `json:"domain" validate:"required"`
}

// SecretRef references an external secret (FRK-INFRA-011/060).
type SecretRef struct {
	ARN string `json:"arn,omitempty"`
	Env string `json:"env,omitempty"`
	Key string `json:"key" validate:"required"`
}

// Validate checks a typed resource struct (FRK-INFRA-010).
func Validate(v any) error {
	if err := validate.Struct(v); err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	return nil
}
