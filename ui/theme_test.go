package ui

import (
	"strings"
	"testing"
)

func TestThemeScriptDefaults(t *testing.T) {
	s := string(Theme{}.Script())
	for _, want := range []string{
		"<script>",
		"krewire-theme",
		"localStorage",
		"(prefers-color-scheme: dark)",
		"data-theme-toggle",
		"auto",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("default script missing %q", want)
		}
	}
}

func TestThemeScriptCustom(t *testing.T) {
	s := string(Theme{StorageKey: "site-theme", Default: "dark"}.Script())
	if !strings.Contains(s, `k="site-theme"`) {
		t.Errorf("script missing custom storage key: %q", s)
	}
	if !strings.Contains(s, `d="dark"`) {
		t.Errorf("script missing custom default: %q", s)
	}
}

func TestThemeButton(t *testing.T) {
	b := string(Theme{}.Button())
	for _, want := range []string{
		`data-theme-toggle`,
		`aria-label="Toggle light/dark theme"`,
		`icon-sun`,
		`icon-moon`,
	} {
		if !strings.Contains(b, want) {
			t.Errorf("button missing %q", want)
		}
	}
}

func TestThemeStorageKeyAndDefault(t *testing.T) {
	if got := (Theme{}).storageKey(); got != "krewire-theme" {
		t.Errorf("default storage key = %q", got)
	}
	if got := (Theme{Default: "bogus"}).defaultTheme(); got != "auto" {
		t.Errorf("invalid default = %q, want auto", got)
	}
	if got := (Theme{Default: "light"}).defaultTheme(); got != "light" {
		t.Errorf("default = %q, want light", got)
	}
}
