package ui

import (
	"strings"
	"testing"
)

func TestDefaultPalettesComplete(t *testing.T) {
	for name, p := range map[string]Palette{
		"light": DefaultLightPalette,
		"dark":  DefaultDarkPalette,
	} {
		vars := p.cssVars(Palette{})
		for _, token := range []string{
			"base-1", "base-1-content",
			"base-2", "base-2-content",
			"base-3", "base-3-content",
			"primary", "primary-content",
			"secondary", "secondary-content",
			"accent", "accent-content",
			"ghost", "ghost-content",
			"neutral", "neutral-content",
			"success", "success-content",
			"info", "info-content",
			"warning", "warning-content",
			"error", "error-content",
		} {
			if !strings.Contains(vars, "--"+token+":") {
				t.Errorf("%s palette missing --%s", name, token)
			}
		}
	}
}

func TestPaletteMergeAndEmit(t *testing.T) {
	p := Palette{Primary: "#123456"}
	vars := p.cssVars(DefaultLightPalette)
	if !strings.Contains(vars, "--primary: #123456;") {
		t.Errorf("override not emitted: %q", vars)
	}
	if !strings.Contains(vars, "--base-1: #fdfdfb;") {
		t.Errorf("default not merged: %q", vars)
	}
}

func TestThemeStyleEmitsPalette(t *testing.T) {
	s := string(Theme{}.Style())
	for _, want := range []string{
		"<style>",
		":root{--base-1",
		`:root[data-theme="dark"]{--base-1`,
		"(prefers-color-scheme: dark)",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("style missing %q", want)
		}
	}
}

func TestThemeScriptIncludesPalette(t *testing.T) {
	s := string(Theme{Light: Palette{Primary: "#7c3aed"}}.Script())
	if !strings.Contains(s, "<style>") {
		t.Error("script must embed the palette style")
	}
	if !strings.Contains(s, "--primary: #7c3aed;") {
		t.Errorf("custom light primary not emitted: %q", s)
	}
	if !strings.Contains(s, `<script>`+"(function(){var k=") {
		t.Error("script must still contain the theming runtime")
	}
}
