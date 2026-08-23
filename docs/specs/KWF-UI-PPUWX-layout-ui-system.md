# Specification — Krewire Layout & UI System

| Field       | Value                                       |
| ----------- | ------------------------------------------- |
| SpecID      | KWF-PPUWX                              |
| Title       | Krewire Layout & UI System                   |
| Status      | Draft                                       |
| Date        | 2026-08-18                                  |
| Author      | Krewire Contributors                         |
| Domain      | Frameworks — UI / Web                       |

## 1. Context

The current SSG system (`framework/web/ssg`) defines layouts, components, and pages declaratively in `krewire.yaml` (or `ssg.yaml`). While powerful, this approach has limitations:

- Layouts are tied to a specific YAML config file, making them non-reusable across projects
- No standard layout primitives (header, footer, sidebar, grid, etc.) — each project reinvents them
- Hard to compose layouts programmatically or share layout logic between projects
- The `web/ssg` package mixes layout concerns with static export logic

The Krewire ecosystem needs a **reusable, composable Layout & UI System**. Its
design is deliberately chosen over the existing Go fullstack web frameworks
after studying their failure modes (see §1.1): Revel's reflection magic and
stalled maintenance, Buffalo's coupling to the npm/webpack JS toolchain,
Beego's monolithic all-in-one design, Gin/Echo's lack of any UI layer. Krewire
does not copy or react to these — it offers a strictly better model: an
idiomatic Go-native component, rendered through a compilation-free, explicit
pipeline. It must:

- Live in `framework/ui` (alongside theming)
- Provide standard layout widgets and UI components
- Keep declarative config **markup-free**: `krewire.yaml` carries only routing and
  data props (a page = `path` + `layout` + `root` + `data`), never HTML or widget trees
- Keep widget composition in **project files and Go code**, not in config — so a
  100-page site stays a short config with a reusable widget library
- Can be used both declaratively (via config) and programmatically (in Go)
- Is independent of the SSG/export pipeline — works with `web`, `mdbind`, or custom builders

### 1.1 Design Constraints (survey of existing Go fullstack frameworks)

Four architectures dominate the Go web ecosystem; each was weighed against
Krewire's goals before the model in §2 was chosen:

| Framework | Architecture | Failure mode Krewire avoids | Krewire's answer |
| --------- | ------------ | -------------------------- | --------------- |
| Revel     | Reflection auto-registration, generated `main()`, own CLI build step | "Magic" that hides control flow; errors surface at runtime not build time; project depends on framework tooling | Explicit `Registry.Register` in Go code; `krewire build` only *reads* files, generates nothing |
| Buffalo   | Generators + plush templates + POP ORM + webpack/npm JS build | Two toolchains (Go + JS), template language lock-in, scaffolding that can't be inspected | Pure Go, no node_modules; stdlib `html/template`; scaffolding only via `krewire new`, output is plain inspectable files |
| Beego     | Monolithic batteries-included framework | Feels like Rails/Java; steep learning curve; hard to swap internals | `ui`/`web`/`ssg` as small composable packages (toolkit, not monolith) |
| Gin/Echo  | Minimal routing only, no UI layer | Huge impulse for every site to reinvent layout/theming/src in YAML | `ui.Layout` + registry + scoped CSS + theming all ship standard |

The unifying constraint: **Krewire must not inherit the pain points of any
existing framework** — no reflection-based control flow, no custom templating
DSL, no JS toolchain, no generated build artifacts, no per-project `cmd`.

## 2. Problem Statement

Every Krewire project (landing page, docs site, blog, app) reimplements common layout patterns:
- Page shell (header, main, footer)
- Navigation bars with theme toggle
- Content grids, cards, sections
- Sidebar layouts for docs
- Hero sections, CTAs, feature grids

These are currently hardcoded in each project's YAML or Go templates, leading to:
- Duplication of layout logic across projects
- Inconsistent markup and accessibility patterns
- Difficulty sharing improvements (e.g., a better header) across sites
- Tight coupling between layout and the SSG config format
- Markup embedded in `krewire.yaml`, which makes configs long, unreadable, and
  impossible to reuse across pages and projects

## 3. Goals

- **G1** — Provide a `ui.Layout` type that composes a page from reusable regions (header, main, footer, aside, etc.)
- **G2** — Ship standard UI components: `Header`, `Footer`, `Nav`, `Sidebar`, `Container`, `Grid`, `Card`, `Section`, `Hero`, `ThemeToggle`
- **G3** — Enable layout composition via Go code (programmatic) and declarative config
- **G4** — Decouple layout from SSG: `ui` has no dependency on `web/ssg` or static export
- **G5** — Integrate with existing theming: components use CSS variables from `ui.Theme`
- **G6** — Support slots/regions for flexible content injection (e.g., `Header{Left: ..., Center: ..., Right: ...}`)
- **G7** — Provide accessible, semantic HTML5 markup by default
- **G8** — Provide a component registry so pages reference components by name and configure them with data props — declarative config carries data, never markup
- **G9** — Let users define custom components as template files in the project without a per-project `cmd`
- **G10** — Provide a Go-native component model (a `Component` renders typed props to HTML and composes other components) that is explicit, compilation-free, and idiomatic — no reflection-based control flow, no generated build artifacts
- **G11** — Keep `krewire.yaml` stable as a data contract: schema changes only when the user's project structure changes, never for a new page or component
- **G12** — Allow users to replace any layer (registry entry, component, layout, render pipeline) without forking the framework

## 4. Non-Goals

- **NG1** — A full CSS framework (no utility classes, no component variants beyond what's needed for layout)
- **NG2** — JavaScript behavior beyond the existing theme toggle **in this phase**; the component output contract stays JS-framework-agnostic (§5.10, KWF-F2TQC) so client-side JS/TS frameworks can be added later without changing markup
- **NG3** — Replacing `html/template` — components render to `template.HTML` strings
- **NG4** — Runtime layout mutation; layouts are composed at build/render time
- **NG5** — A new templating DSL/parser; authoring stays Go-native (`html/template`) and config-driven (data props)
- **NG6** — Runtime compilation: no generated Go code, `main()`, or build-time code generation step

## 5. Requirements

### 5.1 Layout Primitives

| ID         | Requirement                                                       | Priority |
| ---------- | ----------------------------------------------------------------- | -------- |
| LAY-001    | `ui.Layout` struct with named regions: `Header`, `Main`, `Footer`, `Aside` (sidebar), each accepting `template.HTML` | Must |
| LAY-002    | `Layout.Render() template.HTML` emits a complete `<html>` document with `<head>` (theme script, meta, title) and `<body>`; `Main` is wrapped in `<main data-ui-component="main">` and `Aside` in `<aside data-ui-component="aside">`, while `Header`/`Footer` are emitted as-is since their components own their semantic elements | Must |
| LAY-003    | Regions are optional; omitted regions don't render their wrapper elements | Must |
| LAY-004    | `Layout.WithHead(head template.HTML)` to inject custom `<head>` content (fonts, meta, styles) | Must |
| LAY-005    | `Layout.WithBodyAttrs(attrs template.HTMLAttr)` to add attributes to `<body>` (e.g., `data-page="home"`) | Must |

### 5.2 Standard UI Components

| ID         | Requirement                                                       | Priority |
| ---------- | ----------------------------------------------------------------- | -------- |
| LAY-010    | `ui.Header` — top bar with slots: `Brand`, `Nav`, `Actions` (e.g., theme toggle, GitHub link); renders `<header>` with `nav` inside | Must |
| LAY-011    | `ui.Footer` — bottom bar with slots: `Left`, `Center`, `Right`; renders `<footer>` | Must |
| LAY-012    | `ui.Nav` — navigation links with active state support; renders `<nav><ul><li><a>...</a></li></ul></nav>` | Must |
| LAY-013    | `ui.Container` — max-width wrapper with responsive padding; renders `<div class="container">` | Must |
| LAY-014    | `ui.Grid` — responsive CSS grid with configurable columns/gap; renders `<div class="grid">` | Must |
| LAY-015    | `ui.Card` — bordered container with optional header, body, footer; renders `<article class="card">` | Must |
| LAY-016    | `ui.Section` — vertical rhythm container with optional background; renders `<section>` | Must |
| LAY-017    | `ui.Hero` — prominent top section with headline, subtext, CTA slots, and optional illustration/terminal; renders `<section class="hero">` | Must |
| LAY-018    | `ui.ThemeToggle` — re-exports `ui.Theme.Button` with optional wrapper for consistent placement | Must |
| LAY-019    | `ui.Badge` — inline label with semantic variants (primary, neutral, success, etc.) | Should |
| LAY-020    | `ui.Button` — styled `<a>`/`<button>` with variants (primary, ghost, outline) | Should |

### 5.3 Composition & Slots

| ID         | Requirement                                                       | Priority |
| ---------- | ----------------------------------------------------------------- | -------- |
| LAY-030    | All components accept `template.HTML` slots — callers compose by passing rendered HTML | Must |
| LAY-031    | Helper `ui.Slot(content ...template.HTML) template.HTML` concatenates multiple slot values | Must |
| LAY-032    | Components use scoped CSS via `data-ui-component="name"` (consistent with `web/ssg`'s `data-kiw-component`) | Must |
| LAY-033    | Component styles are exported as `ComponentNameCSS` constants (e.g., `HeaderCSS`, `CardCSS`) for inclusion in the page's stylesheet | Must |

### 5.4 Theming Integration

| ID         | Requirement                                                       | Priority |
| ---------- | ----------------------------------------------------------------- | -------- |
| LAY-040    | All components use CSS variables from `ui.Theme` (`--primary`, `--base-1`, `--radius-card`, etc.) | Must |
| LAY-041    | No hardcoded colors in component CSS — only CSS custom properties | Must |
| LAY-042    | Components respect `--radius-card`, `--transition`, `--max-width` from the theme | Should |

### 5.5 Declarative Config Support

| ID         | Requirement                                                       | Priority |
| ---------- | ----------------------------------------------------------------- | -------- |
| LAY-050    | `ui.LayoutConfig` struct mirroring `ui.Layout` for YAML/JSON decoding | Must |
| LAY-051    | `ui.LayoutFromConfig(cfg *LayoutConfig) *Layout` builds a Layout from config | Must |
| LAY-052    | Config supports referencing named components defined elsewhere in the config | Should |
| LAY-053    | Provide YAML-decodable `NavConfig`, `HeaderConfig`, and `FooterConfig` for the corresponding components | Must |
| LAY-054    | `HeaderConfig` supports a brand slot, a nav reference, raw actions HTML, and a `theme_toggle` flag rendering the layout's theme button | Must |
| LAY-055    | `Layout` carries an optional list of stylesheet URLs emitted as `<link rel="stylesheet">` in the document head | Must |

### 5.6 Package Structure & Dependencies

| ID         | Requirement                                                       | Priority |
| ---------- | ----------------------------------------------------------------- | -------- |
| LAY-060    | All types live in `github.com/krewire/framework/ui` | Must |
| LAY-061    | `ui` imports only standard library + `github.com/krewire/libs/core` (for exit codes if needed) | Must |
| LAY-062    | `ui` does NOT import `framework/web`, `net/http`, or `html/template` beyond `template.HTML` | Must |
| LAY-063    | `framework/web/ssg` can import `ui` for layout/components, but `ui` knows nothing about `ssg` | Must |

### 5.7 Component Registry & Declarative Props

Users must never write markup in `krewire.yaml`. Pages reference components by
name and configure them with data props. The registry maps component names to
Go-native renderers; config carries only data.

| ID         | Requirement                                                       | Priority |
| ---------- | ----------------------------------------------------------------- | -------- |
| LAY-100    | Provide a `Component` interface: `Render(props any) (template.HTML, error)` so Go-native components render from arbitrary props | Must |
| LAY-101    | Provide a `Registry` mapping component names to `Component`s with `Register(name, comp)`, `Get(name)`, and `Render(name, props)` | Must |
| LAY-102    | Ship a default registry pre-loaded with the standard components (`Header`, `Footer`, `Nav`, `Container`, `Grid`, `Card`, `Section`, `Hero`, `Badge`, `Button`, `ThemeToggle`) | Must |
| LAY-103    | Declarative config selects components by name (`root`) and passes a props map (`data`) that the registry renders; config contains no HTML | Must |
| LAY-104    | Props are typed per component (e.g. `HeroProps`, `GridProps`); YAML `data` decodes into the component's props type with unknown keys rejected loudly | Must |
| LAY-105    | Components rendered through the registry still carry their scoped style (`data-ui-component`) and CSS is collected like any component | Must |

### 5.8 Project Component Files

Users author custom components as files in the project, reusing the same
authoring model as built-ins. Files live under a project `ui/` directory and
are loaded by the builder (krewire/`ssg`); projects need no `cmd`. The widget
tree — how a page is composed — lives here and in Go code, **never** in config.

| ID         | Requirement                                                       | Priority |
| ---------- | ----------------------------------------------------------------- | -------- |
| LAY-110    | A project component is a file (e.g. `ui/components/<name>.ui.html`) with a Go `html/template` body and an optional scoped CSS block | Must |
| LAY-111    | `ssg` (or the krewire CLI) discovers project component files, registers each under its base name, and exposes them to `Registry.Render` | Must |
| LAY-112    | Project components compose built-ins and each other using Go template funcs (e.g. `component "name"`); props flow through the same data value | Must |
| LAY-113    | Project component CSS is scoped to its root element and collected into the site stylesheet exactly like built-in component CSS | Must |
| LAY-114    | Files use `html/template` only — no custom DSL; directives and iteration rely on standard Go template actions and registered funcs | Must |
| LAY-115    | The layout shell itself may be authored as a project file (`ui/layouts/<name>.ui.html`) using the same `Layout` region slots | Should |

### 5.9 Config Without Markup

| ID         | Requirement                                                       | Priority |
| ---------- | ----------------------------------------------------------------- | -------- |
| LAY-120    | Deprecate inline HTML component bodies in declarative config (`ComponentConfig.Body`); replacement is registry references + project files | Must |
| LAY-121    | Keep `LayoutConfig.UI` (the ui shell) as the supported declarative layout path; header/footer/nav stay data-driven | Must |
| LAY-122    | `ssg.Page`/`PageConfig` must resolve `Root` through the component registry first, then project files; undefined names fail the build with a clear error | Must |
| LAY-123    | The landing page (`krewire.github.io`) migrates to data-only config + registry components, demonstrating zero markup in `krewire.yaml` | Must |
| LAY-124    | A multi-page site adds pages as short config entries (`path`, `layout`, `root`, `data`) without repeating any widget tree in config; composition stays in the reusable component files | Must |
| LAY-125    | Registry entries and the render pipeline are replaceable: a project may override any registered component by name or provide a custom `Component` for a built-in name without forking the framework | Should |

### 5.10 Framework-Neutral Output (Future JS/TS Readiness)

Zero-JS is the default (KWF-F2TQC), but the output contract must never block a
future client-side JS/TS framework. These requirements pin the seams that make
hydration/islands possible later **without changing server markup or config**.

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| LAY-130     | Component props remain JSON-serializable (per LAY-104); renderers may emit the props as `data-props` JSON on the root element for client-side mounting, opt-in and off by default | Must |
| LAY-131     | The `data-ui-component` scope attribute is the documented, stable mount-point contract; identical props must render identical bytes | Must |
| LAY-132     | Component markup must not assume any specific client framework; no framework-specific data attributes are emitted by default | Must |
| LAY-133     | Component CSS stays separate from markup and must never depend on runtime JavaScript | Must |
| LAY-134     | The theme-toggle inline script remains the sole default JS, namespaced (`window.krewireTheme`) and isolated so it coexists with mounted client frameworks | Must |
| LAY-135     | A project may attach a per-component/page client integration (script slot, mount root) as an additive extension through the registry/layout layer, never by editing built-in output | Should |

## 6. Non-Functional Requirements

- **NFR1** — **Memory safety.** No `unsafe` package.
- **NFR2** — **Zero external deps.** Only Go stdlib.
- **NFR3** — **Accessibility.** Semantic HTML5, proper ARIA attributes, focus management.
- **NFR4** — **Testability.** Every component has a `*_test.go` verifying output structure.
- **NFR5** — **Quality gates.** `gofmt`, `go vet ./...`, `go test ./...` pass.

## 7. Success Criteria

- **S1** — A project builds a complete page using only `ui.Layout` + `ui.Header`/`Footer`/`Hero`/`Grid`/`Card` + `ui.Theme` — no custom layout HTML in the project.
- **S2** — The landing page (`krewire.github.io`) migrates to use `ui.Layout` + components instead of inline layout in `krewire.yaml`, with zero markup in the config.
- **S3** — `mdbind` (book builder) can use `ui.Layout` for its page shell instead of hardcoded templates.
- **S4** — Changing `ui.HeaderCSS` updates the header style across all consuming projects without config changes.
- **S5** — A project can define a custom component as a file under its `ui/` directory and reference it from config by name — no `cmd`, no DSL.
- **S6** — `Registry.Render("Hero", props)` produces the same output as the equivalent built-in component, and unknown names return an error.
- **S7** — `krewire build` produces the landing site without invoking any JS toolchain, generated code, or reflection-based control flow; a project can replace a built-in component by name and have it take effect with zero framework changes.

## 8. Related Specifications

| SpecID      | KWF-PPUWX                              |
| --------- | ----------------------------------------------- |
| [KWF-V0TMZ](./KWF-UI-V0TMZ-web-theming-system.md)        | UI Theming System (provides `ui.Theme`, `ui.Palette`) |
| [KWF-0Z671](./KWF-UI-0Z671-krewire-ui-framework.md)       | Krewire UI Framework (this spec extends it) |
| [KWF-M07QS](./KWF-WEB-M07QS-krewire-web-framework.md)      | Krewire Web Framework (consumer) |
| [KWF-PT8OD](./KWF-SSG-PT8OD-static-site-generator.md)     | Static Site Generator (consumer) |
| [KWF-F2TQC](./KWF-JS-F2TQC-js-ts-framework-integration.md) | JS/TS Framework Integration (future bridge) |
| [KWM-FX9H2](https://github.com/krewire/mdbind/blob/main/docs/specs/KWM-BUILDER-FX9H2-mdbind-site-builder.md) | mdbind Site Builder (consumer) |

## 9. References

- [KWF-CMBZJ](./KWF-META-CMBZJ-krewire-meta-framework.md) — Krewire Meta-Framework initial specification.