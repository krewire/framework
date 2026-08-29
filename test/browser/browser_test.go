// Tests for KWF-TEST-N8R2Q
package browser

import (
	"net/http"
	"testing"

	ftest "github.com/krewire/framework/test"
)

// Spec: KWF-TEST-N8R2Q KWF-TST-N8R-001 Scope: Service
func TestKWF_TST_N8R_001_Browser_New_Valid(t *testing.T) {
	ftest.Spec(t, "KWF-TEST-N8R2Q", "KWF-TST-N8R-001")
	srv := ftest.Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body><nav><a href="/next">next</a></nav><h1>Hello</h1></body></html>`))
	}))
	b := New(t, srv.URL)
	b.Navigate("/").WaitVisible("nav").WaitText("h1", "Hello")
	if got := b.Text("h1"); got != "Hello" {
		t.Errorf("Text h1 = %q want Hello", got)
	}
	b.HTMLAssert().Has("nav a", 1)
	b.Screenshot("browser_test_unit")
}

// Spec: KWF-TEST-N8R2Q KWF-TST-N8R-002 Scope: Service
func TestKWF_TST_N8R_002_Browser_Click_Valid(t *testing.T) {
	ftest.Spec(t, "KWF-TEST-N8R2Q", "KWF-TST-N8R-002")
	srv := ftest.Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/next" {
			_, _ = w.Write([]byte(`<h1>Next</h1>`))
			return
		}
		_, _ = w.Write([]byte(`<a href="/next">next</a>`))
	}))
	b := New(t, srv.URL)
	b.Navigate("/").Click("a").WaitText("h1", "Next")
}
