# framework/test — Krewire Testing Framework

Generic test helpers — works with or without spec-driven. Parent spec: `KWF-TEST-P4R3N`.

Import `github.com/krewire/framework/test` (`ftest` alias recommended).

## HTTP — Fluent Chain (KWF-TEST-H7P4L)

```go
import ftest "github.com/krewire/framework/test"

func TestWeb(t *testing.T) {
    // fluent builder → Do → chainable assertions
    ftest.Do(t, handler, ftest.GET(t, "/").Request()).
        Status(200).
        Contains("Krewire")
}

func TestJSON(t *testing.T) {
    var out struct{ Name string `json:"name"` }
    ftest.Do(t, handler, ftest.JSONRequest(t, "POST", "/api", map[string]string{"name":"a"}).Request()).
        Status(200).
        JSON(&out)
}

func TestWithServer(t *testing.T) {
    srv := ftest.Server(t, handler) // httptest.Server with t.Cleanup
    // use srv.URL for external fetch or browser
}
```

Backward compat: `ftest.NewRequest` / `ftest.Record` / `ftest.EqualStatus` still work.

## UI — HTML-Aware Snapshots (KWF-TEST-U9K3M)

```go
func TestUI(t *testing.T) {
    html := render() // e.g. ui.Layout or web handler
    ftest.HTML(t, html).Has("nav a", 3).HasText("nav", "Docs")
    ftest.Snapshot(t, "nav_golden", html) // testdata/nav_golden.golden, UPDATE_GOLDEN=1 to refresh
    ftest.GoldenHTML(t, "home", html)     // alias for HTML golden
}
```

Normalization strips whitespace and hashed `assets/*.hash.css` → `assets/*.css`.

## Browser — Headless (KWF-TEST-N8R2Q, opt-in)

```go
import "github.com/krewire/framework/test/browser"

func TestBrowser(t *testing.T) {
    browser.SkipIfNoBrowser(t) // skip unless TEST_BROWSER=1
    srv := ftest.Server(t, handler)
    b := browser.New(t, srv.URL)
    b.Navigate("/").WaitVisible("nav").Click("a").WaitText("h1", "Hello")
    b.Screenshot("home") // testdata/screenshots/home.html (png when chromedp active)
    b.HTMLAssert().Has("nav", 1)
}
```

Helpers: `Equal`, `NoError`, `HasError`, `Contains`, `NotContains`, `True`, `TempDir`, `ReadFile`, `AssertFile`, `WriteFile`, `Golden`, `Snapshot`, `GoldenHTML`, `HTML`, `Spec` (optional), `Request`/`GET`/`POST`/`JSONRequest`, `Do`, `Server`, `CookieJar`, `WithCookies`, `NewRequest`, `Record`, `EqualStatus`. No `libs/core` dependency — generic Go `testing`.

Run `TEST_BROWSER=1 go test ./...` to include browser tests; otherwise they are skipped.
