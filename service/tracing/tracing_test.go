// Tests for KWF-L5H2F
package tracing

import (
	"context"
	"testing"
)

// Spec: KWF-L5H2F FRK-SVC-040 Scope: Unit
func TestNew_ValidConfig(t *testing.T) {
	tr, err := New(Config{ServiceName: "test", Exporter: "otlp"})
	if err != nil {
		t.Fatal(err)
	}
	if tr == nil {
		t.Fatal("expected non-nil tracer")
	}
}

// Spec: KWF-L5H2F FRK-SVC-040 Scope: Unit
func TestNew_EmptyServiceName(t *testing.T) {
	_, err := New(Config{Exporter: "otlp"})
	if err == nil {
		t.Fatal("expected error for empty serviceName")
	}
}

// Spec: KWF-L5H2F FRK-SVC-040 Scope: Unit
func TestNoopTracer(t *testing.T) {
	tr := NoopTracer{}
	ctx := context.Background()
	ctx2, span := tr.StartSpan(ctx, "test")
	if ctx2 == nil {
		t.Fatal("expected non-nil context")
	}
	span.SetAttribute("key", "value")
	span.RecordError(nil)
	span.End()
	if err := tr.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
}

// Spec: KWF-L5H2F FRK-SVC-042 Scope: Unit
func TestNew_AllExporters(t *testing.T) {
	for _, exp := range []string{"otlp", "jaeger", "zipkin", "stdout"} {
		_, err := New(Config{ServiceName: "test", Exporter: exp})
		if err != nil {
			t.Fatalf("exporter %q: %v", exp, err)
		}
	}
}
