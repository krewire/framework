# Specification — Krewire UI Theming System

| Field       | Value                                       |
| ----------- | ------------------------------------------- |
| SpecID      | KWF-V0TMZ                              |
| Title       | Krewire UI Theming System                    |
| Status      | Draft                                       |
| Date        | 2026-08-18                                  |
| Author      | Krewire Contributors                         |
| Domain      | Frameworks — UI                             |

## 1. Context

The theming system is shipped from the Krewire UI Framework (`framework/ui`) as
`ui.Theme`, `ui.Palette`, and `ui.Color`. It emits an inline script and a
toggle button for static sites. Sites follow the system color scheme by
default, let users switch between light and dark, and remember the choice
without any server round-trip.

## 2. Problem Statement

Light/dark support is re-implemented per site with ad-hoc scripts, inconsistent
storage keys, and a visible flash of the wrong theme. The framework provides no
shared, testable primitive, so every consumer — including mdbind — reinvents
the same toggle.

## 3. Goals

- G1 — Provide a `Theme` type with light/dark/auto preferences.
- G2 — Persist the user's choice across visits.
- G3 — Apply the theme before first paint, avoiding a flash of the wrong mode.
- G4 — Ship a ready-made toggle button wired by the script.
- G5 — Let sites configure the full color palette for light and dark modes.

## 4. Non-Goals

- NG1 — Server-side theming, cookies, or themes beyond light/dark.
- NG2 — Theme-aware image generation or color transformation.
- NG3 — Runtime palette editing; palettes are compiled into the site at build time.

## 5. Requirements

| ID         | Requirement                                                       | Priority |
| ---------- | ----------------------------------------------------------------- | -------- |
| FRK-TH-001 | Support `auto`, `light`, and `dark` preferences, with `auto` following the system color scheme. | Must |
| FRK-TH-002 | Persist the user's choice in `localStorage` under a configurable key. | Must |
| FRK-TH-003 | Apply the theme synchronously from an inline script so no wrong-theme flash occurs. | Must |
| FRK-TH-004 | Expose a toggle on `window.krewireTheme` and wire any element with `data-theme-toggle`. | Must |
| FRK-TH-005 | Provide ready-made toggle button markup (`Theme.Button`).          | Must |
| FRK-TH-006 | Keep the `color-scheme` meta element in sync with the applied theme. | Must |
| FRK-TH-007 | Provide a configurable palette (`ui.Palette`) for light and dark modes with the base, primary, secondary, accent, ghost, neutral, and semantic colors, each with a `*-content` companion. | Must |
| FRK-TH-008 | The palette is emitted as CSS custom properties (`--base-1`, `--primary-content`, …) scoped to `:root` for light and `:root[data-theme="dark"]` for dark. | Must |
| FRK-TH-009 | Empty palette fields fall back to the shipped default palettes, so sites may override a single color. | Must |
| FRK-TH-010 | Ship complete `DefaultLightPalette` and `DefaultDarkPalette` values covering all tokens. | Must |

## 6. Non-Functional Requirements

- NFR1 — **Memory safety.** The `unsafe` package must not be used.
- NFR2 — **Dependency-free.** No external JavaScript or CSS libraries.
- NFR3 — **Portability.** Works in evergreen browsers; degrades gracefully to the system scheme.
- NFR4 — **Quality gates.** `gofmt`, `go vet ./...`, and `go test ./...` must pass.

## 7. Success Criteria

- S1 — `Theme{}.Script()` and `Theme{}.Button()` render embeddable markup.
- S2 — The script reads `localStorage`, follows the system scheme for `auto`, and wires `data-theme-toggle`.
- S3 — A site built with the theme renders before first paint without a wrong-theme flash.
- S4 — Setting `Theme{Light: Palette{Primary: "…"}}` overrides the token in the emitted style while the rest falls back to defaults.

## 8. Related Specifications

| SpecID      | KWF-V0TMZ                              |
| --------- | ----------------------------------------------- |
| [KWF-0Z671](./KWF-UI-0Z671-krewire-ui-framework.md) | Krewire UI Framework (ships this system) |
| [KWF-M07QS](./KWF-WEB-M07QS-krewire-web-framework.md) | Krewire Web Framework             |
| [KWM-FX9H2](https://github.com/krewire/mdbind/blob/main/docs/specs/KWM-BUILDER-FX9H2-mdbind-site-builder.md) | mdbind Site Builder |

## 9. References

- [KWF-CMBZJ](./KWF-META-CMBZJ-krewire-meta-framework.md) — Krewire Framework initial specification.