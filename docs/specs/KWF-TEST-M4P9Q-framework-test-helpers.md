# Specification — Framework Test Helpers (MVP)

| Field  | Value |
| ------ | ----- |
| SpecID | KWF-TEST-M4P9Q |
| Title  | Framework Test Helpers — MVP (`framework/test`) |
| Status | Draft |
| Date   | 2026-08-23 |
| Author | Krewire Contributors |
| Domain | Framework — Testing |

## 1. Context

Krewire is spec-driven and workspace-driven. Existing tests are scattered: `framework/web` uses `net/http/httptest` directly, `framework/web/ssg` uses ad-hoc `assertFile` helpers, `framework/tui` duplicates `testApp` setup, `framework/ui` has no snapshot helper. Contributors copy-paste `assertFile`, `testApp`, `strings.Contains` checks. There is no shared, tiny, zero-deps test helper that is idiomatic Go, spec-traceable, and covers the 9 workloads (web, SSG, tui, ui, app) without pulling `testify`.

The ecosystem needs a minimal `framework/test` package that is stdlib-only (except `libs/core` for spec types), zero-cost when not imported, and that makes `KWL-TEST-P8M4L` compliance trivial: tagging a test with `Spec: KWL-XXX` and asserting with helpers that produce clear failures.

## 2. Problem Statement

- **Current pain:** No generic `framework/test` helpers exist — each package reimplements `Equal`, `Contains`, `Golden`, and `httptest` boilerplate with inconsistent `t.Helper()` and golden handling (`UPDATE_GOLDEN`). `KWL-TEST-P8M4L` traceability is unenforced.
- **Affected consumers:** `framework` contributors, `framework/web`/`ui`/`mdbind` authors, and reviewers auditing spec-to-test links.
- **Cost of leaving unsolved:** Test helpers drift per package, `gofmt`/`go vet` diverge, and spec IDs cannot be `grep`-found, so coverage stays invisible and onboarding repeats helpers.

## 3. Goals

- G1 — Single import `github.com/krewire/framework/test` gives all MVP helpers; zero deps beyond stdlib + `libs/core` (optional).
- G2 — Covers the most common framework test patterns: equality, `NoError`, HTTP handler testing, file/golden snapshots for SSG/UI, temp-dir helpers.
- G3 — Spec-traceability helper satisfies `KWL-TEST-P8M4L` (`// Tests for` + `Spec(...)` call) and is grep-friendly.
- G4 — Idiomatic: helpers take `*testing.T` and call `t.Helper()`, use `cmp` not reflection magic, work with table-driven tests.
- G5 — Backward compatible: existing `TestFoo` tests keep passing; adoption is opt-in.

## 4. Non-Goals

- NG1 — Not a testify replacement (no `assert` package with 50 functions); only ~10 helpers for MVP.
- NG2 — Not a new test runner; still `go test ./...` via `kiw test`.
- NG3 — Not covering `framework/runtime` (WASM), `framework/service`/`worker`/`infra` in MVP (future).

## 5. Requirements

### 4.1 Package & Scope

| ID | Requirement | Priority |
|----|-------------|----------|
| KWF-TST-M4P-001 | Package `test` at `framework/test` (`import "github.com/krewire/framework/test"`), Go 1.22, `gofmt` clean, `go vet` clean, no external deps beyond stdlib. May optionally import `github.com/krewire/libs/core` for spec types but must not require it at runtime. | Must |
| KWF-TST-M4P-002 | Scope `Package` — helpers live in the same package for reuse; no separate `testutil` needed. | Must |

### 4.2 Core Assertions (MVP)

| ID | Requirement | Priority |
|----|-------------|----------|
| KWF-TST-M4P-010 | `Equal(t, got, want any)` — reports `got != want` with `%q` diff and `t.Helper()`. | Must |
| KWF-TST-M4P-011 | `NoError(t, err)` and `HasError(t, err)` — fail with `err` if unexpected. | Must |
| KWF-TST-M4P-012 | `Contains(t, s, substr string)` and `NotContains(t, s, substr string)` — for HTML/SSG assertions. | Must |
| KWF-TST-M4P-013 | `True(t, cond bool, msg string)` — generic boolean. | Should |

### 4.3 HTTP Helpers (web)

| ID | Requirement | Priority |
|----|-------------|----------|
| KWF-TST-M4P-020 | `NewRequest(t, method, target string, body io.Reader) *http.Request` — helper that fails on error. | Must |
| KWF-TST-M4P-021 | `Record(handler http.Handler, req *http.Request) *httptest.ResponseRecorder` — executes handler and returns recorder. | Must |
| KWF-TST-M4P-022 | `EqualStatus(t, rec, want int)` — checks `rec.Code == want`. | Must |

### 4.4 File & Golden Helpers (SSG/ui)

| ID | Requirement | Priority |
|----|-------------|----------|
| KWF-TST-M4P-030 | `TempDir(t) string` — wrapper around `t.TempDir()` with `t.Helper()`. | Should |
| KWF-TST-M4P-031 | `ReadFile(t, path string) string` and `AssertFile(t, path, want string)` — for SSG `Export` tests. | Must |
| KWF-TST-M4P-032 | `Golden(t, name, got string)` — compares `got` to `testdata/<name>.golden`; with `UPDATE_GOLDEN=1` rewrites golden. Uses `GOOS=js` guard? Not needed for MVP (plain file). | Should |

### 4.5 Spec Traceability

| ID | Requirement | Priority |
|----|-------------|----------|
| KWF-TST-M4P-040 | `Spec(t, specID, reqID string)` — logs `Spec: <specID> <reqID>` and is grep-friendly; calls `t.Helper()`. Enables `rg "Spec:"` audit. | Must |
| KWF-TST-M4P-041 | Test file header for helpers themselves: `// Tests for KWF-TEST-M4P9Q` and each test function contains RequirementID in name (e.g., `TestKWF_TST_M4P_010_Equal_Valid`). | Must |

### 4.6 Example & Docs

| ID | Requirement | Priority |
|----|-------------|----------|
| KWF-TST-M4P-050 | Provide `framework/test/README.md` with 10-line example for HTTP and Golden. | Should |
| KWF-TST-M4P-051 | Migrate one existing test (e.g., `framework/web/ssg` or `framework/tui`) to use `framework/test` helpers as reference (non-breaking). | Should |

## 5. Non-Functional

- NFR1 — stdlib-only, `gofmt`/`go vet`/`go test` green.
- NFR2 — Zero-cost when not imported.
- NFR3 — Docs in English, Markdown, spec-driven.

## 7. Success Criteria

- S1 — `go test ./framework/test -count=1` passes; `gofmt -l .` empty.
- S2 — One migrated test uses `ftest.Equal`/`ftest.Contains` and `ftest.Spec`.
- S3 — `rg "KWF-TST-M4P"` finds spec and tests.

## 7. Related Specs

| SpecID | Title |
|--------|-------|
| KWL-TEST-P8M4L | Spec-Driven Testing |
| KWL-ARCH-J2K9Q | Ecosystem Scope Levels |
| KWF-M8K2Q | Unified Framework Vision |
