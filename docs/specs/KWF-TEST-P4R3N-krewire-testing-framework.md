# Specification — Krewire Testing Framework (`framework/test`)

| Field  | Value |
| ------ | ----- |
| SpecID | KWF-TEST-P4R3N |
| Title  | Krewire Testing Framework (`framework/test`) |
| Status | Draft |
| Date   | 2026-08-26 |
| Author | Krewire Contributors |
| Domain | Framework — Testing |

## 1. Context

`framework/test` MVP (`KWF-TEST-M4P9Q`) delivers ~10 helpers (`Equal`, `Contains`, `NewRequest`/`Record`, `Golden`, `Spec`) that make `KWL-TEST-P8M4L` compliance trivial. It is the only test package that is generic, stdlib-only, and `gofmt` clean.

Three workload families now need first-class testing but have no unified surface: `framework/web` HTTP (routes, middleware, cookies, redirects), `framework/ui` + `framework/web/ssg` UI (HTML, scoped CSS, `Theme`), and `framework/runtime` browser (WASM hydration, interactive islands). Each family currently vendors its own ad-hoc assertions, and E2E is either skipped or pulls `chromedp` ad-hoc.

This spec is the **parent** for the Krewire Testing Framework. It does not ship helpers itself; it defines the shared contract, package layout, and progression that the three child specs implement: HTTP, UI, and Browser. It keeps the core `framework/test` stdlib-only while making `browser` an opt-in battery skipped when Chrome is absent.

## 2. Problem Statement

- **Current pain:** No single `framework/test` surface covers `web` + `ui` + `runtime` — each family vendors ad-hoc `httptest`, `strings.Contains`, or `chromedp` snippets, duplicating lifecycle (`t.Cleanup`, `httptest.Server`) and golden handling (`UPDATE_GOLDEN`). Reviewers cannot trace a UI `nav` check to a `FRK-*` row.
- **Affected consumers:** Framework maintainers, `framework/ui`/`web` contributors, doc authors validating `html/template` + `Theme`, and students copying ad-hoc patterns into production tests.
- **Cost of leaving unsolved:** Test helpers drift per package, spec coverage `KWL-TEST-P8M4L` stays unenforced, browser E2E is either skipped or pulls `chromedp` without a shared skip contract, and onboarding a new workload requires re-inventing assertions instead of importing `ftest`.

## 3. Goals

- G1 — `framework/test` is the single entry point for all testing needs; child specs are progressive batteries (HTTP → UI → Browser).
- G2 — Every helper is `t.Helper()`-aware, `gofmt`/`go vet` clean, and `Spec`-traceable (`KWF-TEST-M4P9Q` + `KWL-TEST-P8M4L`).
- G3 — Core package stays stdlib-only and zero-cost when not imported; `browser` is the only subpackage that may import `chromedp`.
- G4 — Spec traceability: every `Must` has a test `// Tests for <SpecID>` with `RequirementID` in the name.

## 4. Non-Goals

- NG1 — Not a `testify` replacement; only helpers that earn their keep for Krewire workloads.
- NG2 — Not a new test runner (`go test ./...` via `kiw test` stays).
- NG3 — Not covering `framework/service`/`worker`/`infra` integration tests.

## 5. Requirements

### 4.1 Package & Scope

| ID | Requirement | Scope | Priority |
|----|-------------|-------|----------|
| KWF-TFW-P4R-001 | Package `test` at `framework/test` remains `import "github.com/krewire/framework/test"`; browser helpers live in `framework/test/browser` to keep the core stdlib-only. | Module | Must |
| KWF-TFW-P4R-002 | Go 1.22+, `gofmt`/`go vet` clean; core has no `chromedp` dep; `browser` may import `chromedp` optionally and must `t.Skip` when Chrome is unavailable. | Module | Must |
| KWF-TFW-P4R-003 | Every helper calls `t.Helper()` and is usable with or without `ftest.Spec(t, ...)`. | Unit | Must |

### 4.2 Child Specs

| ID | Requirement | Scope | Priority |
|----|-------------|-------|----------|
| KWF-TFW-P4R-010 | HTTP Testing Framework child spec (`KWF-TEST-H7P4L`) defines fluent HTTP chain (`Request` → `Do` → `Response`) and must be implemented before UI depends on it. | Module | Must |
| KWF-TFW-P4R-011 | UI Testing Framework child spec (`KWF-TEST-U9K3M`) defines HTML-aware assertions (`HTML`, `Snapshot`) and theme helpers. | Module | Must |
| KWF-TFW-P4R-012 | Browser Testing Framework child spec (`KWF-TEST-N8R2Q`) defines headless browser (`chromedp`) lifecycle in `framework/test/browser`, opt-in via `TEST_BROWSER=1`. | Service | Must |
| KWF-TFW-P4R-013 | README at `framework/test/README.md` must document the three layers with 3 examples (HTTP fluent, UI snapshot, browser guarded) and remain `go vet` clean. | Module | Must |

## 6. Non-Functional Requirements

- NFR1 — **Stdlib-first, zero-cost:** core `framework/test` has no `chromedp`/`testify` dep.
- NFR2 — **Idiomatic Go, English docs, Markdown, spec-driven.**
- NFR3 — `gofmt -l .` empty, `go test ./...` green in `framework` with `TEST_BROWSER=""`.

## 7. Success Criteria

- S1 — Child specs exist and are indexed: `rg "KWF-TEST-H7P4L|KWF-TEST-U9K3M|KWF-TEST-N8R2Q"` finds them.
- S2 — `go test ./framework/test -count=1` passes for HTTP + UI; `TEST_BROWSER=1 go test ./framework/test/browser -count=1` passes or gracefully skips when chrome absent.
- S3 — `rg "KWF-TFW-P4R"` finds parent spec and its tests.

## 8. Related Specifications

| SpecID | Title |
|--------|-------|
| KWF-TEST-M4P9Q | Framework Test Helpers — MVP (`framework/test`) |
| KWF-TEST-H7P4L | HTTP Testing Framework |
| KWF-TEST-U9K3M | UI Testing Framework |
| KWF-TEST-N8R2Q | Browser Testing Framework |
| KWL-TEST-P8M4L | Spec-Driven Testing |
| KWL-ARCH-J2K9Q | Ecosystem Scope Levels |
| KWF-M8K2Q | Unified Framework Vision |

## 9. References

- `KWF-TEST-M4P9Q` MVP helpers
- `KWF-WEB-P3V8X` Expressive HTTP
