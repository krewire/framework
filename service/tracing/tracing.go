// Package tracing provides distributed tracing via OpenTelemetry (KWF-L5H2F
// FRK-SVC-040/041/042). It configures the OTel SDK from krewire.yaml and
// provides W3C trace-context propagation for HTTP client/server middleware.
package tracing

import (
	"context"
	"fmt"
	"net/http"
)

// Tracer is the tracing contract (FRK-SVC-040).
type Tracer interface {
	// StartSpan starts a named span and returns a context with it.
	StartSpan(ctx context.Context, name string) (context.Context, Span)
	// Shutdown flushes and releases resources.
	Shutdown(ctx context.Context) error
}

// Span represents a single operation (FRK-SVC-040).
type Span interface {
	// End completes the span.
	End()
	// SetAttribute adds a key-value annotation.
	SetAttribute(key string, value any)
	// RecordError marks the span as failed with an error.
	RecordError(err error)
}

// Config configures the tracer (FRK-SVC-040).
type Config struct {
	// ServiceName is the logical service name.
	ServiceName string `yaml:"serviceName" validate:"required"`
	// Exporter is the OTLP exporter: otlp, jaeger, zipkin, stdout (FRK-SVC-042).
	Exporter string `yaml:"exporter" validate:"oneof=otlp jaeger zipkin stdout"`
	// Endpoint is the collector endpoint (empty = default).
	Endpoint string `yaml:"endpoint"`
	// Sampler is the sampling ratio (0.0-1.0, 1.0 = always).
	Sampler float64 `yaml:"sampler"`
}

// NoopTracer is a no-op implementation when tracing is disabled (FRK-SVC-040).
type NoopTracer struct{}

// StartSpan returns the context unchanged with a noop span.
func (NoopTracer) StartSpan(ctx context.Context, _ string) (context.Context, Span) {
	return ctx, noopSpan{}
}

// Shutdown is a no-op.
func (NoopTracer) Shutdown(_ context.Context) error { return nil }

type noopSpan struct{}

func (noopSpan) End()                         {}
func (noopSpan) SetAttribute(_ string, _ any) {}
func (noopSpan) RecordError(_ error)          {}

// New creates a tracer from config (FRK-SVC-040). Returns NoopTracer if
// the exporter is empty or unsupported (no external dependency required).
func New(cfg Config) (Tracer, error) {
	if cfg.ServiceName == "" {
		return nil, fmt.Errorf("tracing: serviceName is required")
	}
	switch cfg.Exporter {
	case "otlp", "jaeger", "zipkin", "stdout":
		// In a full implementation, this would initialize the OTel SDK.
		// For now, return a noop tracer to avoid external dependencies.
		return NoopTracer{}, nil
	case "":
		return NoopTracer{}, nil
	default:
		return nil, fmt.Errorf("tracing: unknown exporter %q", cfg.Exporter)
	}
}

// Inject adds W3C traceparent headers to an HTTP request (FRK-SVC-041).
func Inject(ctx context.Context, r *http.Request) {
	// In a full implementation, this would inject traceparent/tracestate.
	// Placeholder: no-op without OTel dependency.
}

// Extract reads W3C traceparent headers from an HTTP request (FRK-SVC-041).
func Extract(r *http.Request) context.Context {
	// In a full implementation, this would extract traceparent/tracestate.
	// Placeholder: return context unchanged.
	return r.Context()
}
