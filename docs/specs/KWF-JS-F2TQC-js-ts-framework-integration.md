# Specification — JS/TS Framework Integration (Future)

| Field       | Value                                       |
| ----------- | ------------------------------------------- |
| SpecID      | KWF-F2TQC                              |
| Title       | JS/TS Framework Integration — Future Bridge |
| Status      | Draft                                       |
| Date        | 2026-08-19                                  |
| Author      | Krewire Contributors                         |
| Domain      | Frameworks — Web / Client Integration       |

## 1. Context

Krewire ships **zero JavaScript by default**: pages are pure server-rendered HTML
+ CSS (KWF-PT8OD, KWF-0F2EB), and the only inline script is the isolated theme
toggle. This keeps output fast, secure, and dependency-free — and that default
stays.

The ecosystem, however, must remain **open to all modern JS/TS frameworks**
(React, Vue, Svelte, Astro, Solid, Qwik, Angular, …) without committing to any
one of them. This spec is the single statement of how the framework stays ready
for that future: a **framework-agnostic bridge contract** baked into the render
model, so a client-side framework can be layered on later without changing
server markup, config, or the existing pipelines. It is the "escape hatch"
(G12 of KWF-PPUWX) for interactivity.

## 2. Problem Statement

If the framework hardcodes its HTML output without a stable contract, adding any
JS/TS framework later means re-rendering pages, changing component output, or
forking the pipeline — the exact coupling Krewire rejects (KWF-PPUWX §1.1,
Buffalo's npm lock-in). Conversely, if the contract only "exists in spirit", no
real integration will ever fit. The framework needs explicit, documented seams:

- Stable DOM + scope attributes that a client framework can query reliably.
- Serialized props so a client component can re-render from the same data the
  server used (no double-fetch, no data duplication).
- CSS that never depends on runtime JS.
- A defined place where a future client build can attach scripts/mount points.
- A default script set that never conflicts with client frameworks.

## 3. Goals

- G1 — Keep zero-JS-by-default as the only shipped default; nothing in the bridge is emitted unless a project opts in.
- G2 — Define a framework-agnostic contract (scope attrs + `data-props` + deterministic HTML + CSS/JS separation) that any JS/TS framework can target.
- G3 — Guarantee the bridge is **additive**: existing static export, SSR, and API pipelines stay byte-identical without it.
- G4 — Keep the framework's dependency on JS tooling at zero at compile time; client builds are the project's concern (served as assets).
- G5 — Share one data contract between server props and the API JSON (KWF-230KF) so client-side data fetching and hydration use the same shape.

## 4. Non-Goals

- NG1 — Bundling or compiling JS/TS inside the framework, or adopting a JS package manager (no node_modules).
- NG2 — Committing to any specific JS/TS framework, island system, or build tool.
- NG3 — Server-side rendering of client frameworks in this phase.
- NG4 — Changing the default zero-JS output or the theme-toggle script.
- NG5 — A codegen step that generates framework-specific glue code.

## 5. Requirements

### 5.1 Render Contract (must hold in every pipeline today)

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| JSF-001     | Every rendered component root carries a stable `data-ui-component="<name>"` attribute; scope attributes are the documented mount-point contract for client frameworks. | Must |
| JSF-002     | HTML output for identical props is deterministic (same bytes), so client-side diffing/hydration is predictable. | Must |
| JSF-003     | Component markup must not depend on a specific client framework; no framework-specific data attributes are emitted by default. | Must |
| JSF-004     | All component CSS is emitted separately (collected `style.css`) and must never rely on runtime JavaScript. | Must |
| JSF-005     | The theme-toggle is the only inline script by default; it is namespaced (`window.krewireTheme`), side-effect-isolated, and must not mutate or assume document structure owned by other frameworks. | Must |

### 5.2 Props & Data Bridge

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| JSF-010     | Component props must remain JSON-serializable (already enforced by strict JSON decode in KWF-PPUWX LAY-104); rendering may optionally emit the props as `data-props` JSON on the root element for client mounting. | Must |
| JSF-011     | The `data-props` JSON shape uses the same field naming as the API JSON contract (KWF-230KF), so hydration data and fetched data are interchangeable. | Should |
| JSF-012     | Props emission is opt-in per component/page and off by default; enabling it must not change the rendered HTML except for the `data-props` attribute. | Must |

### 5.3 Client Integration Seam

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| JSF-020     | The render pipeline exposes a no-op extension hook by default (e.g. a per-page script/mount slot) that a future client layer can fill without touching server rendering. | Should |
| JSF-021     | Layouts/pages must have a defined injection point for `<script>` tags and mount roots (`<div id="app">`-style), served as static assets via `App.Static`/SSG assets. | Must |
| JSF-022     | A project may ship compiled client bundles (JS/TS output) under `public/` or `site/` and serve them; the framework must not transform, cache-bust, or block them. | Must |
| JSF-023     | The future client layer plugs in as a registry/layout additive extension (KWF-PPUWX §5.10, KWF-0F2EB FRK-SRV-042), never by forking the pipeline (G12). | Must |

### 5.4 Documentation & Testing

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| JSF-030     | Provide a reference integration guide per framework (React, Vue, Svelte, Astro, Solid) when the bridge lands, targeting the §5.1 contract. | Should |
| JSF-031     | Tests must pin that default output contains zero `<script>` (except theme toggle) and remains byte-identical with and without the bridge present. | Must |

## 6. Non-Functional Requirements

- NFR1 — **Memory safety.** The `unsafe` package must not be used.
- NFR2 — **Dependencies.** This spec adds no framework dependency; zero JS tooling at compile time.
- NFR3 — **Performance.** Emitting `data-props` must not duplicate large data trees unless requested; default output unchanged.
- NFR4 — **Quality gates.** `gofmt`, `go vet ./...`, `go test ./...` pass.

## 7. Success Criteria

- S1 — A React, Vue, or Svelte component can mount on a rendered Krewire page using only `data-ui-component` + `data-props`, with no server markup change.
- S2 — Static export and SSR output remain byte-identical to pre-bridge builds (default mode).
- S3 — The theme toggle works alongside a mounted client framework without conflicts.
- S4 — API JSON and `data-props` share one documented field-naming contract.
- S5 — Every new JS/TS framework supported later requires no framework code change — only a project-side integration.

## 8. Related Specifications

| SpecID      | KWF-F2TQC                              |
| --------- | ----------------------------------------------- |
| [KWF-PPUWX](./KWF-UI-PPUWX-layout-ui-system.md) | Layout & UI System (§5.10 framework-neutral output) |
| [KWF-PT8OD](./KWF-SSG-PT8OD-static-site-generator.md) | Static Site Generator (hydratable-ready export) |
| [KWF-0F2EB](./KWF-WEB-0F2EB-server-frontend-pipeline.md) | Server & Frontend Rendering Pipeline (SSR seam) |
| [KWF-230KF](./KWF-HTTP-230KF-http-api-pipeline.md) | HTTP & API Pipeline (shared JSON contract) |
| [KWF-C4087](./KWF-APP-C4087-krewire-app-framework.md) | Krewire App Framework (client asset serving) |
| [KWF-M07QS](./KWF-WEB-M07QS-krewire-web-framework.md) | Krewire Web Framework (host) |

## 9. References

- [KWF-PPUWX](./KWF-UI-PPUWX-layout-ui-system.md) §1.1 — design constraints survey (Buffalo npm lock-in avoided).
- Astro "islands architecture" (concept reference only; no adoption implied).