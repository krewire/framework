package web

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/krewire/framework/ui"
	"github.com/krewire/framework/web/ssg"
)

const shellLayout = `<!DOCTYPE html><html><head><title>{{.Title}}</title></head><body>{{.Content}}</body></html>`

func appFixture() *App {
	a := NewApp()
	a.Theme(&ui.Theme{})
	a.Layout(ssg.Layout{Name: "Shell", Body: shellLayout})
	a.Component(ssg.Component{Name: "Greeting", Body: `<h1>Hello {{.Name}}</h1>`})
	return a
}

func TestAppServesStaticPage(t *testing.T) {
	a := appFixture()
	a.Page(PageSpec{Path: "/", Title: "Home", Layout: "Shell", Root: "Greeting", Data: map[string]any{"Name": "Krewire"}})

	rec := do(a.Handler(), http.MethodGet, "/", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Hello Krewire") {
		t.Errorf("body missing component output: %q", body)
	}
	if !strings.Contains(body, "<title>Home</title>") {
		t.Errorf("body missing layout title: %q", body)
	}
}

func TestAppDynamicPageUsesRequestData(t *testing.T) {
	a := appFixture()
	a.Page(PageSpec{
		Path: "/greet", Title: "Greet", Layout: "Shell", Root: "Greeting",
		DataFunc: func(r *http.Request) (any, error) {
			return map[string]any{"Name": r.URL.Query().Get("name")}, nil
		},
	})

	rec := do(a.Handler(), http.MethodGet, "/greet?name=Budi", "")
	if !strings.Contains(rec.Body.String(), "Hello Budi") {
		t.Errorf("dynamic data must reflect query: %q", rec.Body.String())
	}
}

func TestAppServesPageAndAPIFromOneHandler(t *testing.T) {
	a := appFixture()
	a.Page(PageSpec{Path: "/", Layout: "Shell", Root: "Greeting", Data: map[string]any{"Name": "x"}})
	a.Router().Get("/api/ping", func(w http.ResponseWriter, _ *http.Request, _ Params) {
		JSON(w, http.StatusOK, map[string]string{"pong": "true"})
	})

	rec := do(a.Handler(), http.MethodGet, "/", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("page status = %d", rec.Code)
	}
	rec = do(a.Handler(), http.MethodGet, "/api/ping", "")
	if !strings.Contains(rec.Body.String(), `"pong":"true"`) {
		t.Errorf("api body = %q", rec.Body.String())
	}
}

func TestAppExportByteMatchesSSR(t *testing.T) {
	a := appFixture()
	a.Page(PageSpec{Path: "/", Title: "Home", Layout: "Shell", Root: "Greeting", Data: map[string]any{"Name": "Krewire"}})
	a.Page(PageSpec{Path: "/about", Title: "About", Layout: "Shell", Root: "Greeting", Data: map[string]any{"Name": "About"}})

	out := filepath.Join(t.TempDir(), "site")
	if _, err := a.Export(out); err != nil {
		t.Fatalf("Export: %v", err)
	}

	live := do(a.Handler(), http.MethodGet, "/", "")
	want, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != live.Body.String() {
		t.Error("exported index.html differs from live SSR render")
	}
	about, err := os.ReadFile(filepath.Join(out, "about", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(about), "Hello About") {
		t.Errorf("about export = %q", about)
	}
}

func TestAppExportSkipsDynamicPages(t *testing.T) {
	a := appFixture()
	a.Page(PageSpec{Path: "/", Layout: "Shell", Root: "Greeting", Data: map[string]any{"Name": "s"}})
	a.Page(PageSpec{
		Path: "/dyn", Layout: "Shell", Root: "Greeting",
		DataFunc: func(*http.Request) (any, error) { return map[string]any{"Name": "dyn"}, nil },
	})

	out := filepath.Join(t.TempDir(), "site")
	if _, err := a.Export(out); err != nil {
		t.Fatalf("Export: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "dyn", "index.html")); !os.IsNotExist(err) {
		t.Errorf("dynamic page must not be exported")
	}
}

func TestAppEmitProps(t *testing.T) {
	a := appFixture()
	a.Page(PageSpec{Path: "/", Layout: "Shell", Root: "Greeting", Data: map[string]any{"Name": "x"}, EmitProps: true})

	body := do(a.Handler(), http.MethodGet, "/", "").Body.String()
	if !strings.Contains(body, `data-props="{`) {
		t.Errorf("page must emit data-props: %q", body)
	}

	a2 := appFixture()
	a2.Page(PageSpec{Path: "/", Layout: "Shell", Root: "Greeting", Data: map[string]any{"Name": "x"}})
	body2 := do(a2.Handler(), http.MethodGet, "/", "").Body.String()
	if strings.Contains(body2, "data-props") {
		t.Errorf("default output must not emit data-props: %q", body2)
	}
}

func TestAppScriptsInjected(t *testing.T) {
	a := appFixture()
	a.Page(PageSpec{Path: "/", Layout: "Shell", Root: "Greeting", Data: map[string]any{"Name": "x"}, Scripts: []string{"/assets/app.js"}})

	body := do(a.Handler(), http.MethodGet, "/", "").Body.String()
	if !strings.Contains(body, `<script src="/assets/app.js"></script></body>`) {
		t.Errorf("scripts not injected before </body>: %q", body)
	}
}

func TestAppServesSiteAssetsLive(t *testing.T) {
	a := appFixture()
	a.Page(PageSpec{Path: "/", Layout: "Shell", Root: "Greeting", Data: map[string]any{"Name": "x"}})

	for _, path := range []string{"/assets/ui.css", "/assets/theme.css"} {
		rec := do(a.Handler(), http.MethodGet, path, "")
		if rec.Code != http.StatusOK {
			t.Errorf("%s not served: %d", path, rec.Code)
			continue
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/css") {
			t.Errorf("%s content-type = %q", path, ct)
		}
	}
}

func TestAppStaticFS(t *testing.T) {
	a := appFixture()
	a.Static("/assets", fstest.MapFS{
		"app.js":    &fstest.MapFile{Data: []byte("console.log(1)")},
		"style.css": &fstest.MapFile{Data: []byte("body{}")},
		"sub/x.txt": &fstest.MapFile{Data: []byte("x")},
	})

	rec := do(a.Handler(), http.MethodGet, "/assets/app.js", "")
	if rec.Code != http.StatusOK || rec.Body.String() != "console.log(1)" {
		t.Errorf("static fs file not served: %d %q", rec.Code, rec.Body.String())
	}
	rec = do(a.Handler(), http.MethodGet, "/assets/sub/x.txt", "")
	if rec.Body.String() != "x" {
		t.Errorf("nested fs file = %q", rec.Body.String())
	}
}

func TestAppThemeAssetsCollected(t *testing.T) {
	a := appFixture()
	a.Theme(&ui.Theme{})
	a.Page(PageSpec{Path: "/", Layout: "Shell", Root: "Greeting", Data: map[string]any{"Name": "x"}})

	out := filepath.Join(t.TempDir(), "site")
	if _, err := a.Export(out); err != nil {
		t.Fatalf("Export: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "assets", "theme.css")); err != nil {
		t.Errorf("theme.css missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "assets", "ui.css")); err != nil {
		t.Errorf("ui.css missing: %v", err)
	}
}

func TestAppRenderErrorIs500(t *testing.T) {
	a := NewApp()
	a.Layout(ssg.Layout{Name: "Shell", Body: shellLayout})
	a.Component(ssg.Component{Name: "Broken", Body: `{{if}}`})
	a.Page(PageSpec{Path: "/", Layout: "Shell", Root: "Broken", Data: map[string]any{}})

	rec := do(a.Handler(), http.MethodGet, "/", "")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 on render error", rec.Code)
	}
}
