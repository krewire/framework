package test

import (
	"reflect"
	"strings"
	"testing"
)

func Equal(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Equal: got %q want %q", got, want)
	}
}

func NoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("NoError: unexpected error %v", err)
	}
}

func HasError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("HasError: expected error, got nil")
	}
}

func Contains(t *testing.T, s, substr string) {
	t.Helper()
	if !contains(s, substr) {
		t.Errorf("Contains: %q does not contain %q", s, substr)
	}
}

func NotContains(t *testing.T, s, substr string) {
	t.Helper()
	if contains(s, substr) {
		t.Errorf("NotContains: %q should not contain %q", s, substr)
	}
}

func True(t *testing.T, cond bool, msg string) {
	t.Helper()
	if !cond {
		t.Errorf("True: %s", msg)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
