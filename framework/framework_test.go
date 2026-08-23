package framework

import (
	"strings"
	"testing"
)

func TestBanner(t *testing.T) {
	b := Banner()
	if !strings.Contains(b, Name) {
		t.Errorf("Banner() = %q, want it to contain %q", b, Name)
	}
	if !strings.Contains(b, Version) {
		t.Errorf("Banner() = %q, want it to contain %q", b, Version)
	}
}
