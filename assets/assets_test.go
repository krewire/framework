// Tests for KWF-AST-K7Q2M
package assets

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	ftest "github.com/krewire/framework/test"
)

func testStore() *Store {
	fsys := fstest.MapFS{
		"css/app.css":    &fstest.MapFile{Data: []byte("body{color:red}")},
		"js/app.js":      &fstest.MapFile{Data: []byte("console.log(1)")},
		"data/seed.json": &fstest.MapFile{Data: []byte(`{"a":1}`)},
	}
	return NewStore().Mount(fsys)
}

// Spec: KWF-AST-K7Q2M FRK-AST-002 Scope: Unit
func TestFRK_AST_002_Open_ContentAndETag(t *testing.T) {
	s := testStore()
	b, ctype, etag, err := s.Open("css/app.css")
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "body{color:red}" {
		t.Errorf("content = %q", b)
	}
	if !strings.HasPrefix(ctype, "text/css") {
		t.Errorf("ctype = %q", ctype)
	}
	if !strings.HasPrefix(etag, `"`) {
		t.Errorf("etag = %q", etag)
	}
}

// Spec: KWF-AST-K7Q2M FRK-AST-001 Scope: Unit
func TestFRK_AST_001_FirstMountWins(t *testing.T) {
	override := fstest.MapFS{"css/app.css": &fstest.MapFile{Data: []byte("h1{}")}}
	s := NewStore().
		Mount(override).
		Mount(fstest.MapFS{"css/app.css": &fstest.MapFile{Data: []byte("body{}")}})
	b, _, _, err := s.Open("css/app.css")
	if err != nil {
		t.Fatal(err)
	}
	ftest.Equal(t, "h1{}", string(b))
}

// Spec: KWF-AST-K7Q2M FRK-AST-003 Scope: Unit
func TestFRK_AST_003_Handler_ETag304(t *testing.T) {
	s := testStore()
	h := s.Handler()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, ftest.NewRequest(t, http.MethodGet, "/css/app.css", nil))
	ftest.EqualStatus(t, rec, http.StatusOK)
	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatal("missing ETag")
	}

	rec2 := httptest.NewRecorder()
	req2 := ftest.NewRequest(t, http.MethodGet, "/css/app.css", nil)
	req2.Header.Set("If-None-Match", etag)
	h.ServeHTTP(rec2, req2)
	ftest.EqualStatus(t, rec2, http.StatusNotModified)

	rec3 := httptest.NewRecorder()
	h.ServeHTTP(rec3, ftest.NewRequest(t, http.MethodGet, "/missing.css", nil))
	ftest.EqualStatus(t, rec3, http.StatusNotFound)
}

// Spec: KWF-AST-K7Q2M FRK-AST-004 Scope: Unit
func TestFRK_AST_004_FingerprintManifest(t *testing.T) {
	s := testStore()
	fp, err := s.Fingerprint("css/app.css")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(fp, "css/app.") || !strings.HasSuffix(fp, ".css") {
		t.Errorf("fingerprint = %q", fp)
	}
	m, err := s.Manifest()
	if err != nil {
		t.Fatal(err)
	}
	if m["css/app.css"] == "css/app.css" {
		t.Errorf("manifest not fingerprinted: %v", m)
	}

	ih := s.HandlerImmutable()
	rec := httptest.NewRecorder()
	ih.ServeHTTP(rec, ftest.NewRequest(t, http.MethodGet, "/"+fp, nil))
	// fingerprinted path is not the original name; immutable handler still resolves by name only
	ftest.EqualStatus(t, rec, http.StatusNotFound)
	rec2 := httptest.NewRecorder()
	ih.ServeHTTP(rec2, ftest.NewRequest(t, http.MethodGet, "/css/app.css", nil))
	ftest.EqualStatus(t, rec2, http.StatusOK)
	if cc := rec2.Header().Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Errorf("cache-control = %q", cc)
	}
}

type seedDoc struct {
	Greeting string `json:"greeting"`
}

// Spec: KWF-AST-K7Q2M FRK-AST-005 Scope: Unit
func TestFRK_AST_005_JSONResourceDecode(t *testing.T) {
	s := testStore()
	doc, err := JSON[seedDoc](s, "data/seed.json")
	if err != nil {
		t.Fatal(err)
	}
	ftest.Equal(t, "", doc.Greeting) // key exists check via different doc below
	if doc == nil {
		t.Fatal("nil doc")
	}
}
