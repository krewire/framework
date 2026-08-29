# Specification — Krewire Browser Testing Framework (`framework/test/browser`)

| Field  | Value |
| ------ | ----- |
| SpecID | KWF-TEST-N8R2Q |
| Title  | Krewire Browser Testing Framework (`framework/test/browser`) |
| Status | Draft |
| Date   | 2026-08-26 |
| Author | Krewire Contributors |
| Domain | Framework — Testing |

## 1. Context

`framework/runtime` (WASM islands, hydration) and docs sites need real browser assertions — navigate, wait for hydrated, click, read text — yet the repo has zero headless-browser primitive. Teams skip E2E or vendor `chromedp` ad-hoc without a shared lifecycle (`httptest.Server` + browser lifecycle + `t.Cleanup`).

This spec adds an opt-in `framework/test/browser` package that is skipped when Chrome is absent, so `go test ./...` stays green in CI without `chromedp`.

## 2. Problem Statement

- **Current pain:** No shared headless browser primitive — `framework/runtime` hydration and docs sites need `Navigate`/`WaitVisible`/`Click`/`Text`/`Screenshot` but repo has zero. Teams either skip E2E or vendor `chromedp` ad-hoc without `httptest.Server` + `t.Cleanup` lifecycle and without a skip contract (`TEST_BROWSER`), so CI is flaky or green by omission.
- **Affected consumers:** `framework/runtime` island authors, doc site maintainers, `framework/test/browser` reviewers who must trust hydration.
- **Cost of leaving unsolved:** WASM hydration regressions (missing `data-theme`, unhydrated island) ship to prod because no `WaitVisible("nav")` gate; `chromedp` is pulled inconsistently, increasing `go.mod` churn and CI flakes.

## 3. Goals

- G1 — `import "github.com/krewire/framework/test/browser"` gives `Browser` with `Navigate`/`WaitVisible`/`Click`/`Text`/`HTML`/`Screenshot`, lifecycle is `t.Helper()` + `t.Cleanup()` safe.
- G2 — Graceful skip when Chrome or `TEST_BROWSER=1` is not set — never fails CI by default.
- G3 — Bridges to `framework/test.HTML` for DOM assertions after navigation.

## 4. Non-Goals

- NG1 — Not covering `framework/service`/`worker`/`infra` integration tests.
- NG2 — Not a new test runner; still `go test ./...`.

## 5. Requirements

| ID | Requirement | Scope | Priority |
|----|-------------|-------|----------|
| KWF-TST-N8R-001 | Package `browser` at `framework/test/browser` with `Browser` type: `New(t, serverURL string) *Browser` starts `chromedp` allocator (`chromedp.NewExecAllocator` + `chromedp.NewContext`), registers `t.Cleanup(cancel)`. If chrome not found or `TEST_BROWSER` not set, `t.Skip("browser: chrome not available")` — never fails. | Service | Must |
| KWF-TST-N8R-002 | `Browser` methods (all `t.Helper()`): `Navigate(path string)`, `WaitVisible(selector string)`, `WaitText(selector, want string)`, `Click(selector string)`, `Text(selector string) string`, `HTML() string`, `Screenshot(name string)` (saves to `testdata/<name>.png`). Each waits with `context.WithTimeout(5s)` and reports selector on failure. | Service | Must |
| KWF-TST-N8R-003 | `Browser.HTMLAssert() *HTMLAssert` — bridges to `framework/test.HTML` for DOM assertions after navigation. | Service | Should |
| KWF-TST-N8R-004 | Package-level helper `SkipIfNoBrowser(t)` for `TestMain` or per-test skip: `if os.Getenv("TEST_BROWSER") != "1" { t.Skip }`. | Service | Should |
| KWF-TST-N8R-005 | Every helper calls `t.Helper()` and is `Spec`-traceable. | Unit | Must |

## 6. Non-Functional Requirements

- NFR1 — **Opt-in, zero-cost:** core `framework/test` has no `chromedp` dep; `browser` subpackage may import `chromedp` via `go.mod` optional dep but must skip gracefully.
- NFR2 — **Idiomatic Go, English docs, Markdown, spec-driven.**
- NFR3 — `gofmt`/`go vet` clean; `go test ./framework/test/browser -count=1` passes or skips when chrome absent.

## 7. Success Criteria

- S1 — `TEST_BROWSER=1 go test ./framework/test/browser -run TestBrowser -count=1` navigates a `httptest.Server` serving a `framework/ui` page, clicks a button, and asserts text — skipped gracefully when chrome absent.
- S2 — `rg "KWF-TST-N8R"` finds spec and tests; `gofmt -l .` empty.

## 8. Related Specifications

| SpecID | Title |
|--------|-------|
| KWF-TEST-P4R3N | Krewire Testing Framework (`framework/test`) — Parent |
| KWF-TEST-M4P9Q | Framework Test Helpers — MVP |
| KWL-TEST-P8M4L | Spec-Driven Testing |
| KWF-T4X9P | WASM Client Runtime (browser hydration) |

## 9. References

- `chromedp` — https://github.com/chromedp/chromedp
- `httptest.Server` lifecycle — https://pkg.go.dev/net/http/httptest#Server
