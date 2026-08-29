// Tests for KWF-TEST-U9K3M
package test

import (
	"os"
	"path/filepath"
	"testing"
)

// Spec: KWF-TEST-U9K3M KWF-TST-U9K-010 Scope: Unit
func TestKWF_TST_U9K_010_HTML_Valid(t *testing.T) {
	Spec(t, "KWF-TEST-U9K3M", "KWF-TST-U9K-010")
	html := `<nav><a class="link" href="/">a</a><a class="link" href="/b">b</a></nav><div id="main">Hello Docs</div>`
	HTML(t, html).Has("nav a", 2).Has("a.link", 2).Has("#main", 1).ContainsText("Hello").HasText("div", "Hello Docs").Attr("a", "href", "/")
}

// Spec: KWF-TEST-U9K3M KWF-TST-U9K-011 Scope: Unit
func TestKWF_TST_U9K_011_Snapshot_Valid(t *testing.T) {
	Spec(t, "KWF-TEST-U9K3M", "KWF-TST-U9K-011")
	name := "ui_snapshot_test"
	path := filepath.Join("testdata", name+".golden")
	_ = os.Remove(path)
	t.Setenv("UPDATE_GOLDEN", "1")
	Snapshot(t, name, `<div>  hello  </div><link href="assets/app.abc123.css">`)
	t.Setenv("UPDATE_GOLDEN", "")
	Snapshot(t, name, `<div>  hello  </div><link href="assets/app.abc123.css">`)
	_ = os.Remove(path)
}

// Spec: KWF-TEST-U9K3M KWF-TST-U9K-012 Scope: Unit
func TestKWF_TST_U9K_012_ThemeSnapshot_Valid(t *testing.T) {
	Spec(t, "KWF-TEST-U9K3M", "KWF-TST-U9K-012")
	name := "theme_snapshot_test"
	path := filepath.Join("testdata", name+".golden")
	_ = os.Remove(path)
	t.Setenv("UPDATE_GOLDEN", "1")
	ThemeSnapshot(t, name, `<html data-theme="dark"><style>:root{--primary:#00c853}</style></html>`)
	t.Setenv("UPDATE_GOLDEN", "")
	ThemeSnapshot(t, name, `<html data-theme="dark"><style>:root{--primary:#00c853}</style></html>`)
	_ = os.Remove(path)
}
