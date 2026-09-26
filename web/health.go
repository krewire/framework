package web

import "github.com/krewire/libs/sec"

// Check is a health check function. Return nil if healthy, or an error if unhealthy.
type Check = sec.Check

// HealthOptions configures health and readiness probe endpoints.
type HealthOptions = sec.HealthOptions

var (
	// WithLivenessPath sets the liveness endpoint path.
	WithLivenessPath = sec.WithLivenessPath
	// WithReadinessPath sets the readiness endpoint path.
	WithReadinessPath = sec.WithReadinessPath
	// WithCheck registers a named health check on readiness probes.
	WithCheck = sec.WithCheck
)

// Health returns a middleware providing /healthz and /readyz probe endpoints.
func Health(opts ...func(*HealthOptions)) Middleware {
	return Middleware(sec.Health(opts...))
}
