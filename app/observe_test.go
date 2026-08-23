package app

import (
	"io"
	"log/slog"
	"strings"
	"testing"
)

func TestTracerHooks(t *testing.T) {
	tr := &recordingTracer{}
	c := New(WithTracer(tr))
	_ = Provide[*db](c, func() *db { return &db{dsn: "x"} })
	_, err := Resolve[*db](c)
	if err != nil {
		t.Fatal(err)
	}
	ev := tr.events()
	if len(ev) < 2 {
		t.Fatalf("expected before/after events, got %v", ev)
	}
	if !strings.HasPrefix(ev[0], "before:") || !strings.HasPrefix(ev[1], "after:") {
		t.Fatalf("unexpected event order: %v", ev)
	}
}

func TestTraceLogging(t *testing.T) {
	var buf strings.Builder
	l := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	c := New(WithLogger(l), WithTrace())
	_ = Provide[*db](c, func() *db { return &db{dsn: "x"} })
	_, err := Resolve[*db](c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "resolve") {
		t.Fatalf("expected trace record, got %q", buf.String())
	}
}

func TestBootSummary(t *testing.T) {
	c, err := NewApp(&countingProvider{}).Build()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve[*counting](c); err != nil {
		t.Fatal(err)
	}
	if got := c.Resolved(); len(got) != 2 {
		t.Fatalf("expected 2 resolved types, got %v", got)
	}
}

func TestResolveErrorIs(t *testing.T) {
	re := &ResolveError{Type: "x", Err: io.ErrUnexpectedEOF}
	if !strings.Contains(re.Error(), "unexpected EOF") {
		t.Fatalf("error must unwrap cause: %q", re.Error())
	}
}
