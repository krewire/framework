# Specification — Krewire UI Framework

| Field       | Value                                       |
| ----------- | ------------------------------------------- |
| SpecID      | KWF-0Z671                              |
| Title       | Krewire UI Framework                         |
| Status      | Draft                                       |
| Date        | 2026-08-18                                  |
| Author      | Krewire Contributors                         |
| Domain      | Frameworks — UI                             |

## 1. Context

The Krewire UI Framework (`framework/ui`) is the shared styling layer for the
Krewire ecosystem. It owns the reusable, configurable styling primitives —
the light/dark theming system, color palettes, and the theme toggle styles —
so that consumers (sites and builders such as `framework/web` and mdbind)
reuse them instead of hardcoding theme markup and CSS.

## 2. Problem Statement

Styling primitives currently live in `framework/web`. Because web is a
routing/rendering layer, every consumer that wants theming (mdbind, the
documentation site, the landing page) imports web even when it needs no
routing. Toggle markup, mode variables, and toggle CSS are duplicated or
hardcoded in each consumer, which defeats reusability and lets the pieces
drift apart.

## 3. Goals

- G1 — Move `Theme`, `Palette`, and `Color` out of `framework/web` into `framework/ui`.
- G2 — Make `ui` the single owner of theming; `web` must not depend on theming types.
- G3 — Ship the theme-mode variables and toggle button styles as exported values so consumers stop hardcoding them.
- G4 — Keep `ui` free of routing and of the `net/http`/`html` coupling found in web.

## 4. Non-Goals

- NG1 — New theme behavior beyond what the theming system already defines.
- NG2 — Component library (cards, navbars, buttons beyond the theme toggle).
- NG3 — A CSS framework; `ui` supplies tokens and small primitives, not a full design system.

## 5. Requirements

| ID         | Requirement                                                       | Priority |
| ---------- | ----------------------------------------------------------------- | -------- |
| FRK-UI-001 | `ui` must provide `Theme`, `Palette`, and `Color` with the same API previously shipped by `web`. | Must |
| FRK-UI-002 | `ui` must own `Theme.Style`, `Theme.Script`, and `Theme.Button` for light/dark/auto theming. | Must |
| FRK-UI-003 | `ui` must export `ThemeModeVarsCSS` declaring `--show-sun`/`--show-moon` for light, dark, and system preference. | Must |
| FRK-UI-004 | `ui` must export `ThemeToggleCSS` styling the `theme-toggle` button from the palette custom properties. | Must |
| FRK-UI-005 | Consumers must not hardcode toggle markup or toggle CSS; they must reuse the `ui` exports. | Must |
| FRK-UI-006 | `ui` must not import `framework/web`, `net/http`, or the `html` parser. | Must |

## 6. Non-Functional Requirements

- NFR1 — **Memory safety.** The `unsafe` package must not be used.
- NFR2 — **Dependency-free.** No external JavaScript or CSS libraries.
- NFR3 — **Quality gates.** `gofmt`, `go vet ./...`, and `go test ./...` must pass.

## 7. Success Criteria

- S1 — `ui.Theme`, `ui.Palette`, and `ui.Color` compile and pass their tests after moving out of web.
- S2 — `framework/web` no longer exports theming types and still builds.
- S3 — A consumer builds a themed site with the toggle styles sourced entirely from `ui` exports.
- S4 — The `framework/ui` package imports no web or routing packages.

## 8. Related Specifications

| SpecID      | KWF-0Z671                              |
| --------- | ----------------------------------------------- |
| [KWF-V0TMZ](./KWF-UI-V0TMZ-web-theming-system.md) | Web Theming System (moved to ui) |
| [KWF-M07QS](./KWF-WEB-M07QS-krewire-web-framework.md) | Krewire Web Framework             |
| [KWF-PT8OD](./KWF-SSG-PT8OD-static-site-generator.md) | Static Site Generator            |
| [KWM-FX9H2](https://github.com/krewire/mdbind/blob/main/docs/specs/KWM-BUILDER-FX9H2-mdbind-site-builder.md) | mdbind Site Builder |

## 9. References

- [KWF-CMBZJ](./KWF-META-CMBZJ-krewire-meta-framework.md) — Krewire Framework initial specification.
