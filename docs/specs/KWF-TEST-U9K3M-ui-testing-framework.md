# Specification — Krewire UI Testing Framework (`framework/test`)

| Field  | Value |
| ------ | ----- |
| SpecID | KWF-TEST-U9K3M |
| Title  | Krewire UI Testing Framework (`framework/test`) |
| Status | Draft |
| Date   | 2026-08-26 |
| Author | Krewire Contributors |
| Domain | Framework — Testing |

## 1. Context

`framework/ui` and `framework/web/ssg` render HTML + scoped CSS + `Theme` vars (`KWF-0Z671`, `KWF-V0TMZ`), but tests assert UI with `strings.Contains(html, "nav")`. No DOM-aware helper exists for `Has("nav a", 3)`, no normalized `Snapshot` that strips whitespace/hashed assets, and no `Theme` snapshot. Golden files are ad-hoc per package and sensitive to asset hashes.

This spec extends `framework/test` with UI-aware helpers that stay stdlib-friendly (via `golang.org/x/net/html`) and honor `UPDATE_GOLDEN=1`, without requiring a browser.

## 2. Problem Statement

- **Current pain:** UI tests use `strings.Contains(html, "nav")` — they miss a missing `</nav>`, count `nav a` incorrectly, break on attribute order, and fail when `assets/app.abc123.css` hash changes. No `Has("nav a", 3)` or `Attr("a","href","/")`, no normalized `Snapshot`, no `ThemeSnapshot` for `data-theme`/`--primary`. Goldens are ad-hoc and whitespace-sensitive.
- **Affected consumers:** `framework/ui`, `framework/web/ssg`, doc site authors, and `framework/test` reviewers who must trust `html/template` + `Theme` output.
- **Cost of leaving unsolved:** Layout regressions (missing `aside`/`header`) and theme regressions (wrong `data-theme`) slip past `Contains`; hashed assets cause flaky goldens; onboarding a new UI component requires copy-pasting `html.Parse` instead of `ftest.HTML`.

## 3. Goals

- G1 — `ftest.HTML(t, html).Has/ContainsText/Snapshot` gives DOM-aware assertions for `framework/ui` and `framework/web/ssg` pages.
- G2 — Snapshots are normalized (trim whitespace, sort attrs, strip `*.hash.css` → `*.css`) and use `Golden` under the hood.
- G3 — Works with or without `ftest.Spec(t, ...)`, `t.Helper()`-aware, clear diffs.

## 3. Non-Goals

- NG1 — Not a browser E2E (covered by `KWF-TEST-N8R2Q`).
- NG2 — Not a CSSOM engine; only HTML structure + text + attrs.

## 4. Requirements

| ID | Requirement | Scope | Priority |
|----|-------------|-------|----------|
| KWF-TST-U9K-001 | Package `test` stays `import "github.com/krewire/framework/test"`; Go 1.22+, `gofmt`/`go vet` clean, stdlib + `golang.org/x/net/html` only. | Module | Must |
| KWF-TST-U9K-010 | `HTML(t, html string) *HTMLAssert` — wraps `golang.org/x/net/html` parse. Methods: `Has(selector string, wantCount int)` (simple CSS: `tag`, `.class`, `#id`, `tag.class`), `HasText(selector, want string)`, `ContainsText(want string)`, `Attr(selector, attr, want string)`, `Count(selector string) int`. Selector engine is tiny (≈80 lines), no external query dep. | Unit | Must |
| KWF-TST-U9K-011 | `Snapshot(t, name, html string)` — normalized snapshot: trims whitespace, sorts attrs, strips hashed `assets/*.hash.css` to `assets/*.css` before golden compare. Uses `Golden(t, name, normalized)`; honors `UPDATE_GOLDEN=1`. | Unit | Must |
| KWF-TST-U9K-012 | `ThemeSnapshot(t, name, html string)` — asserts `ui.Theme` Script/Style presence (`data-theme` + `var(--primary)`) — no visual mismatch between SSR and hydrated markup. | Unit | Should |
| KWF-TST-U9K-013 | `GoldenHTML(t, name, html string)` — alias for `Snapshot` with `.html.golden` extension, for `framework/web/ssg` and `framework/ui` pages. | Unit | Should |
| KWF-TST-U9K-014 | Every helper calls `t.Helper()` and is `Spec`-traceable. | Unit | Must |

## 5. Non-Functional Requirements

- NFR1 — **Stdlib-friendly:** only `golang.org/x/net/html` added; `gofmt`/`go vet`/`go test` green.
- NFR2 — **Hermetic snapshots:** normalized golden does not depend on asset hashes or whitespace.
- NFR3 — **English docs, Markdown, spec-driven.**

## 6. Success Criteria

- S1 — `go test ./framework/test -run TestKWF_TST_U9K -count=1` passes (Has, Snapshot, ThemeSnapshot).
- S2 — One `framework/ui` test migrated to `ftest.HTML(t, html).Has("nav a", 3)`.
- S3 — `rg "KWF-TST-U9K"` finds spec and tests; `gofmt -l .` empty.

## 7. Related Specifications

| SpecID | Title |
|--------|-------|
| KWF-TEST-P4R3N | Krewire Testing Framework (`framework/test`) — Parent |
| KWF-TEST-M4P9Q | Framework Test Helpers — MVP |
| KWL-TEST-P8M4L | Spec-Driven Testing |
| KWF-0Z671 | Krewire UI Framework |
| KWF-V0TMZ | UI Theming System |

## 8. References

- `golang.org/x/net/html` — https://pkg.go.dev/golang.org/x/net/html
- `framework/ui` rendering — `ui/theme.go`
