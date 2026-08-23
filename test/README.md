# framework/test — Test Helpers (MVP)

Import `github.com/krewire/framework/test` (`ftest` alias recommended).

```go
import ftest "github.com/krewire/framework/test"

func TestWeb(t *testing.T) {
    ftest.Spec(t, "KWF-TEST-M4P9Q", "KWF-TST-M4P-020")
    req := ftest.NewRequest(t, "GET", "/", nil)
    rec := ftest.Record(handler, req)
    ftest.EqualStatus(t, rec, 200)
    ftest.Contains(t, rec.Body.String(), "Krewire")
}

func TestGolden(t *testing.T) {
    got := render()
    ftest.Golden(t, "render_home", got) // testdata/render_home.golden
    // UPDATE_GOLDEN=1 go test ./...
}
```

Helpers: `Equal`, `NoError`, `HasError`, `Contains`, `NotContains`, `True`, `TempDir`, `ReadFile`, `AssertFile`, `WriteFile`, `Golden`, `Spec`, `NewRequest`, `Record`, `EqualStatus`.
