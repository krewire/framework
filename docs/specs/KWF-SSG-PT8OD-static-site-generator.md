# Specification — Krewire Static Site Generator

| Field       | Value                                       |
| ----------- | ------------------------------------------- |
| SpecID      | KWF-PT8OD                              |
| Title       | Krewire Static Site Generator                |
| Status      | Draft                                       |
| Date        | 2026-08-18                                  |
| Author      | Krewire Contributors                         |
| Domain      | Frameworks — Web — SSG                      |

## 1. Context

The `web` package provides a low-level routing, templating, and static-export
layer, while `mdbind` generates sites from markdown. Neither offers the
component-based authoring model popularized by Astro and SvelteKit: reusable
components, layouts that wrap page content, and styles scoped to the component
that declares them.

This specification defines an Astro/Svelte-inspired static site generator
(`web/ssg`) that composes `html/template` components and layouts into a static
website with a single build step.

## 2. Problem Statement

Sites built with plain `html/template` repeat navigation, headers, and footers
in every page. Global stylesheets grow unbounded and leak between pages, and
there is no shared notion of a "component" with its own markup and styles.
Every consumer reinvents the same composition and asset-collection logic.

## 3. Goals

- G1 — Model a site as a tree of named components composed through templates.
- G2 — Scope a component's CSS to that component's rendered root element.
- G3 — Support layouts that wrap page content in a shared shell.
- G4 — Produce a complete static site (HTML plus collected CSS) in one build.
- G5 — Serve the same pages during development through a plain HTTP handler.
- G6 — Act as the **static-export mode** of the shared page model, producing output identical to the SSG's dynamic counterpart render.

## 4. Non-Goals

- NG1 — Markdown authoring, collections, or content pipelines (owned by mdbind).
- NG2 — Client-side hydration, islands, or JavaScript frameworks **in this phase**; exported pages must remain hydratable-ready (FRK-SSG-022..024, KWF-F2TQC) so JS/TS frameworks can be layered on without re-export.
- NG3 — Runtime data fetching; all data is provided at build time.

## 5. Requirements

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-SSG-001 | Provide a `Site` builder with named `Component`s (template + scoped style). | Must |
| FRK-SSG-002 | Components compose by invoking other named components within templates. | Must |
| FRK-SSG-003 | Each component's rendered HTML carries a unique scope attribute on its root element, injected automatically. | Must |
| FRK-SSG-004 | Each component's CSS is rewritten so every rule applies only under its scope, with `:root` rules left global. | Must |
| FRK-SSG-005 | Provide `Layout`s with a content slot wrapping page output.         | Must |
| FRK-SSG-006 | Pages declare an output path, title, layout, root component, and data. | Must |
| FRK-SSG-007 | `Site.Build` writes pages as `index.html` (clean URLs) plus assets and a single collected `style.css`. | Must |
| FRK-SSG-008 | Only CSS of components and layouts actually used by built pages is emitted. | Must |
| FRK-SSG-009 | `Site` serves the same pages and assets over HTTP for development.  | Must |
| FRK-SSG-010 | Provide a declarative `Config` describing components, layouts, pages, assets, theme, and site-wide data, decodable from `ssg.yaml`. | Must |
| FRK-SSG-011 | Build a `Site` from a `Config` (`Config.Site`, `BuildFromConfig`); page data merges site-wide data over per-page overrides and injects `Title`, `Description`, and the configured `Theme`. | Must |
| FRK-SSG-012 | Register a trusted-HTML template helper for data fields that intentionally contain markup; all other data remains escaped by default. | Must |
| FRK-SSG-013 | Config theme palettes override the ui defaults by CSS token name (`primary`, `base-1`, …). | Must |
| FRK-SSG-014 | Provide `LayoutFromUI` converting a reusable `ui.Layout` shell into an ssg `Layout`, substituting the ui title/main placeholders with the page's `{{ .Title }}` and `{{ .Content }}` so pages render into the shell; parse errors must surface. | Must |
| FRK-SSG-015 | Declarative config layouts may reference a `ui` shell (`LayoutConfig.UI`); `Config.Site()` builds such layouts through `ui.LayoutFromConfig` + `LayoutFromUI`, merges the site theme when the shell has none, rejects registered `ui` shells with invalid templates, and emits a `assets/ui.css` asset whenever a `ui` shell is used. | Must |
| FRK-SSG-016 | Provide `Config`/`Site` support for the `ui` component registry: a page's `Root` resolves through `ui.Registry` (and project component files) first, then inline config components; config markup (`ComponentConfig.Body`) is deprecated. | Must |
| FRK-SSG-017 | Provide a loader for project component files (`ui/components/*.ui.html`) that registers each under its base name with scoped CSS, so custom components are referenced by name from config. | Must |
| FRK-SSG-018 | Config must remain markup-free: pages are short entries (`path`, `layout`, `root`, `data`) and widget composition lives in component files/Go — never in `krewire.yaml`. | Must |
| FRK-SSG-019 | The site theme/registry resolution is shared with SSR: same `ui.Theme` tokens and component registry; identical input stays byte-identical across build and serve. | Must |
| FRK-SSG-020 | SSG is one mode of the shared page model (KWF-0F2EB): a **static** page exported by the SSG renders through the same component/layout render path as a **dynamic** page served per request. | Must |
| FRK-SSG-021 | Any static page is also servable as a dynamic page by `web.App` with zero markup/texture change, and vice-versa for fixed-data pages. | Must |
| FRK-SSG-022 | Exported pages are hydratable-ready: stable `data-ui-component` scope attributes, deterministic markup, and props optionally serialized as `data-props` JSON (opt-in) for client mounting — without adding any `<script>` to default output. | Must |
| FRK-SSG-023 | Component/layout CSS is exported separately and must never depend on runtime JavaScript; no framework-specific attributes are emitted by default. | Must |
| FRK-SSG-024 | The build must be able to emit a defined script/mount injection point (empty by default) so a future client layer can attach JS/TS bundles as assets without touching server rendering. | Should |

## 6. Non-Functional Requirements

- NFR1 — **Memory safety.** The `unsafe` package must not be used.
- NFR2 — **Minimal dependencies.** HTML parsing uses `golang.org/x/net/html`; no JavaScript or CSS tooling.
- NFR3 — **Quality gates.** `gofmt`, `go vet ./...`, and `go test ./...` must pass.

## 7. Success Criteria

- S1 — A page composed of nested components builds to static HTML where component styles are prefixed by their scope.
- S2 — Building a site with unused components omits their CSS from `style.css`.
- S3 — The development handler renders the same output as the build.
- S4 — A site described entirely by an `ssg.yaml` config builds through `BuildFromConfig` with injected site data, theme, and trusted-HTML helpers.
- S5 — A page whose `Root` names a registered component (built-in or project file) renders from data props with no markup in config.

## 8. Related Specifications

| SpecID      | KWF-PT8OD                              |
| --------- | ----------------------------------------------- |
| [KWF-M07QS](./KWF-WEB-M07QS-krewire-web-framework.md) | Krewire Web Framework             |
| [KWF-V0TMZ](./KWF-UI-V0TMZ-web-theming-system.md) | Krewire Web Theming System       |
| [KWF-PPUWX](./KWF-UI-PPUWX-layout-ui-system.md) | Layout & UI System (registry + Go-native components) |
| [KWF-0F2EB](./KWF-WEB-0F2EB-server-frontend-pipeline.md) | Server & Frontend Rendering Pipeline (shared page model) |
| [KWF-F2TQC](./KWF-JS-F2TQC-js-ts-framework-integration.md) | JS/TS Framework Integration (future bridge) |

## 9. References

- [KWM-FX9H2](https://github.com/krewire/mdbind/blob/main/docs/specs/KWM-BUILDER-FX9H2-mdbind-site-builder.md) — mdbind Site Builder (markdown-driven counterpart).