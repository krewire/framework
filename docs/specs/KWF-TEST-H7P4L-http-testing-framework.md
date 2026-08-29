# Specification — Krewire HTTP Testing Framework (`framework/test`)

| Field  | Value |
| ------ | ----- |
| SpecID | KWF-TEST-H7P4L |
| Title  | Krewire HTTP Testing Framework (`framework/test`) |
| Status | Draft |
| Date   | 2026-08-26 |
| Author | Krewire Contributors |
| Domain | Framework — Testing |

## 1. Context

`framework/test` MVP (`KWF-TEST-M4P9Q`) provides `NewRequest`/`Record`/`EqualStatus` — a thin wrapper over `net/http/httptest`. `framework/web` now spans 9 workloads (expressive routes, CSRF, session, cookie, JWT, cache, ssg) yet tests still hand-roll `httptest.NewRecorder`, parse `Set-Cookie` manually, decode JSON with `if err`, and assert redirects with `if rec.Code !=`. No fluent builder, no cookie/session helpers, no `TestServer` with `baseURL` for `ServeMux` integration. Copy-paste across `framework/web` and `kiw` hides spec coverage (`KWL-TEST-P8M4L`) and keeps `framework/web` tests verbose.

This spec extends the MVP with a fluent HTTP chain that remains stdlib-only and progress-safe.

## 2. Problem Statement

- **Current pain:** Every `framework/web` and `kiw` test reimplements the same 15 lines: `httptest.NewRecorder`, `http.NewRequest`, manual `Set-Cookie` parsing, `if rec.Code != 200`, `json.Unmarshal` with `if err`. No builder for `Header/Cookie/Query/Form/JSON`, no chainable `Status/Contains/JSON`, no `Server` helper with `t.Cleanup`. Copy-paste hides spec traceability (`KWL-TEST-P8M4L`) and makes `framework/web` tests verbose and inconsistent.
- **Affected consumers:** Framework contributors, `mdbind`/`kiw` app authors, `framework/web` reviewers who must audit redirect/cookie/session behavior; students learning `net/http` who copy the hand-rolled pattern into production.
- **Cost of leaving unsolved:** `framework/web` remains the most-copied package, yet its HTTP coverage stays shallow; bugs in `Set-Cookie`/`Location` slip past `strings.Contains` checks; spec coverage `KWF-TST-H7P-*` cannot be enforced because no helper is `t.Helper()`-aware or chainable, so `go test` diffs stay noisy.

## 3. Goals

- G1 — Single import `ftest "github.com/krewire/framework/test"` gives a fluent HTTP chain: request builder → `Do(handler)` → `Response` assertions.
- G2 — Covers the 90% of `framework/web` patterns: method/path/headers/cookies/query/form/JSON/body, status/header/cookie/JSON/redirect assertions, and `httptest.Server` lifecycle.
- G3 — Backward compatible: existing `NewRequest`/`Record`/`EqualStatus` remain and delegate to the new builder (no breaking change).

## 4. Non-Goals

- NG1 — Not covering `framework/service`/`worker`/`infra` integration tests (future).
- NG2 — Not a new test runner; still `go test ./...` via `kiw test`.
- NG3 — Not mandating external `testify` or `chromedp`; HTTP layer stays stdlib-only.

## 5. Requirements

### 4.1 Package & Scope

| ID | Requirement | Scope | Priority |
|----|-------------|-------|----------|
| KWF-TST-H7P-001 | Package `test` at `framework/test` stays `import "github.com/krewire/framework/test"`; Go 1.22+, `gofmt`/`go vet` clean, stdlib-only. | Module | Must |
| KWF-TST-H7P-002 | Every new helper calls `t.Helper()` and is usable with or without `ftest.Spec(t, ...)`. | Unit | Must |

### 4.2 HTTP — Fluent Chain

| ID | Requirement | Scope | Priority |
|----|-------------|-------|----------|
| KWF-TST-H7P-010 | `Request(t) *RequestBuilder` — fluent builder: `Method`, `Path`, `Header`, `Cookie`, `Query`, `Form`, `JSON`, `Body`, `WithContext`. Terminal `Request() *http.Request` and shortcuts `GET(t, path)`, `POST(t, path, body)`, `JSONRequest(t, method, path, v)`. | Unit | Must |
| KWF-TST-H7P-011 | `Do(t, handler, req) *Response` — executes `handler` via `httptest.NewRecorder`, returns `*Response` with chainable assertions. Must accept `http.Handler` and `func(http.ResponseWriter,*http.Request)` via `http.HandlerFunc`. | Unit | Must |
| KWF-TST-H7P-012 | `*Response` assertions (all `t.Helper()`): `Status(want int)`, `Header(key, want string)`, `Contains(text string)`, `NotContains(text string)`, `JSON(v any)` (strict decode, diff on mismatch), `Cookie(name string) *http.Cookie`, `RedirectTo(wantPath string)` (checks `Location` + 3xx), `Body() string`. Each returns `*Response` for chaining, fails immediately on mismatch. | Unit | Must |
| KWF-TST-H7P-013 | `Server(t, handler) *httptest.Server` — starts `httptest.Server` and registers `t.Cleanup(Server.Close)`. Exposes `URL` for browser/external fetch. | Unit | Must |
| KWF-TST-H7P-014 | Cookie/session helpers: `CookieJar`, `WithCookies(req, jar)`, assertions for `Set-Cookie` attributes (`Secure`, `HttpOnly`, `SameSite`). Must parse multiple cookies correctly. | Unit | Should |
| KWF-TST-H7P-015 | Existing `NewRequest`/`Record`/`EqualStatus` remain for backward compat and delegate to the new builder (no breaking change). | Unit | Must |

## 6. Non-Functional Requirements

- NFR1 — **Stdlib-only, zero-cost:** no `chromedp`/`testify` dep; `go test` without new flags.
- NFR2 — **Idiomatic Go:** helpers take `*testing.T`, call `t.Helper()`, return chainable `*Response`, clear `got/want` diffs.
- NFR3 — **English docs, spec-driven:** README example is `go vet` clean.

## 7. Success Criteria

- S1 — `go test ./framework/test -run TestKWF_TST_H7P -count=1` passes (fluent chain, cookie, redirect).
- S2 — One existing `framework/web` test migrated to `ftest.Do(t, handler, ftest.GET(t, "/")).Status(200).Contains("Krewire")`.
- S3 — `rg "KWF-TST-H7P"` finds spec and tests; `gofmt -l .` empty.

## 8. Related Specifications

| SpecID | Title |
|--------|-------|
| KWF-TEST-M4P9Q | Framework Test Helpers — MVP (`framework/test`) |
| KWL-TEST-P8M4L | Spec-Driven Testing |
| KWL-ARCH-J2K9Q | Ecosystem Scope Levels |
| KWF-M8K2Q | Unified Framework Vision |
| KWF-WEB-P3V8X | Expressive HTTP: Routes, Controllers |

## 9. References

- `net/http/httptest` — https://pkg.go.dev/net/http/httptest
- `httptest.Server` lifecycle — https://pkg.go.dev/net/http/httptest#Server
