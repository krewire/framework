package ui

import (
	"strings"
	"testing"

	ftest "github.com/krewire/framework/test"
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

// --- UI Testing Framework (KWF-TEST-U9K3M) — Theme with HTMLAssert & Snapshot ---

// Spec: KWF-TEST-U9K3M KWF-TST-U9K-010 Scope: Unit
func TestKWF_TST_U9K_010_Theme_HTMLAssert(t *testing.T) {
	ftest.Spec(t, "KWF-TEST-U9K3M", "KWF-TST-U9K-010")
	th := Theme{}
	style := string(th.Style())
	ftest.HTML(t, style).Has("style", 1).ContainsText("--primary")
	script := string(th.Script())
	ftest.HTML(t, script).Has("style", 1).Has("script", 1)
	button := string(th.Button())
	ftest.HTML(t, button).Has("button", 1).Attr("button", "data-theme-toggle", "").Has("svg", 2)
}

// Spec: KWF-TEST-U9K3M KWF-TST-U9K-011 Scope: Unit
func TestKWF_TST_U9K_011_Theme_Snapshot(t *testing.T) {
	ftest.Spec(t, "KWF-TEST-U9K3M", "KWF-TST-U9K-011")
	th := Theme{Default: "dark", Light: Palette{Primary: "#111111"}}
	html := string(Layout{Title: "T", Main: "<p>x</p>", Theme: &th}.Render())
	ftest.Snapshot(t, "ui_theme_snapshot", html)
}

// Spec: KWF-TEST-U9K3M KWF-TST-U9K-012 Scope: Unit
func TestKWF_TST_U9K_012_Theme_ThemeSnapshot(t *testing.T) {
	ftest.Spec(t, "KWF-TEST-U9K3M", "KWF-TST-U9K-012")
	th := Theme{}
	ftest.ThemeSnapshot(t, "ui_theme_golden", string(th.Style())+string(th.Script()))
}

// Spec: KWF-TEST-U9K3M KWF-TST-U9K-013 Scope: Unit
func TestKWF_TST_U9K_013_Theme_GoldenHTML(t *testing.T) {
	ftest.Spec(t, "KWF-TEST-U9K3M", "KWF-TST-U9K-013")
	th := Theme{StorageKey: "site-theme", Default: "light"}
	ftest.GoldenHTML(t, "ui_theme_custom", string(th.Script()))
}
