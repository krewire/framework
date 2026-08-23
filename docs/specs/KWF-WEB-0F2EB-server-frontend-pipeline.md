# Specification — Server & Frontend Rendering Pipeline

| Field       | Value                                       |
| ----------- | ------------------------------------------- |
| SpecID      | KWF-0F2EB                              |
| Title       | Server & Frontend Rendering Pipeline        |
| Status      | Draft                                       |
| Date        | 2026-08-19                                  |
| Author      | Krewire Contributors                         |
| Domain      | Frameworks — Web / SSR                      |

## 1. Context

The Krewire ecosystem builds **static** sites (KWF-PT8OD) and, with KWF-230KF,
can expose JSON APIs. The missing piece is **server-side rendering (SSR)**: a
live process that renders the same `ui` components (KWF-PPUWX) into HTML per
request, so a single monolith binary serves frontend pages and JSON APIs from
one `net/http` listener.

`web.App` is that runtime. It composes routes, middleware, pages, static
assets, and the theme/registry shared with the SSG pipeline. Pages are either
**static** (exported, from KWF-PT8OD's model) or **dynamic** (rendered per
request); both feed the same render path so output never diverges.

## 2. Problem Statement

Today Krewire can preview a site (`krewire serve`) and export it, but it cannot
serve a dynamic page or a SPJ-less interactive app: pages are baked at build
time, so anything per-request (auth state, form results, user-owned content,
live dashboards) is impossible without abandoning the framework. A
server-rendered frontend is the missing bridge:

- Dynamic pages need the same component/registry model as static pages.
- A monolith must serve HTML pages, JSON APIs, and assets from one port.
- Lifecycle (startup, graceful shutdown, signal handling) is currently
  reimplemented per app with `http.ListenAndServe`.

`web.App` centralizes these and keeps SSR and SSG on one render path.

## 3. Goals

- G1 — Provide `web.App` composing routes, middleware, pages, static dirs, theme, and registry.
- G2 — Pages render through the shared `ui` component registry + `Layout` shell.
- G3 — Support both **static** pages (exported to files) and **dynamic** pages (rendered per request) with identical output for identical input.
- G4 — Serve HTML, JSON (via KWF-230KF), and embedded assets from one handler.
- G5 — Provide lifecycle helpers: `App.Run(addr)` with graceful shutdown and signal handling.
- G6 — Allow single-binary deployment by mounting assets from an `fs.FS` (embed-friendly).
- G7 — Reuse, not fork: the render path must be the same one the SSG uses so a page can be exported *and* served dynamically.

## 4. Non-Goals

- NG1 — Hot reloading (that's the krewire `dev` command, KWN-6K41E).
- NG2 — Server-side sessions or auth primitives.
- NG3 — SPA client-side rendering or JS hydration **in this phase**; SSR output must stay JS-framework-agnostic and expose an opt-in hydration seam (FRK-SRV-040..043, KWF-F2TQC).
- NG4 — Multiple web servers / cluster management.
- NG5 — A template DSL beyond `html/template` (see KWF-PPUWX).
- NG6 — Replacing KWF-PT8OD: the SSG remains the static-export authority; `App` reuses its model.

## 5. Requirements

### 5.1 App Assembly

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-SRV-001 | Provide `web.NewApp()` returning an `*App`.                       | Must     |
| FRK-SRV-002 | `App.Method(path, pattern, h)` registers API/HTML handlers using the `web` router. | Must |
| FRK-SRV-003 | `App.Use(mw ...Middleware)` adds middleware to every route.       | Must     |
| FRK-SRV-004 | `App.Page(spec PageSpec)` registers a named page with a root component, layout, and data. | Must |
| FRK-SRV-005 | `App.Static(urlPrefix string, fsys fs.FS)` serves files from an embed-compatible FS. | Must |
| FRK-SRV-006 | `App.Theme(t *ui.Theme)` attaches theming; `App.Registry(r *ui.Registry)` overrides the default component registry. | Must |

### 5.2 Page Model (Static/Dynamic)

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-SRV-010 | `PageSpec` names a `Root` component (via the registry) and a `Layout`, mirroring the SSG `PageConfig`. | Must |
| FRK-SRV-011 | A **dynamic** page's data comes from a per-request function `Data func(*http.Request) (any, error)` executed before render. | Must |
| FRK-SRV-012 | A **static** page's data is fixed and renders identically to SSG output for the same component/layout/data. | Must |
| FRK-SRV-013 | `App.Export(outDir string) ([]string, error)` writes static pages as `index.html` plus assets, identical to KWF-PT8OD semantics. | Must |
| FRK-SRV-014 | Render unifies through one internal `renderPage(name, data)` shared by SSR and Export; successful dynamic render equals the exported file minus dynamic data. | Must |
| FRK-SRV-015 | A page render error returns 500 (or the error's mapped status via FRK-H3QD8-API-021). | Must |

### 5.3 Serving & Lifecycle

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-SRV-020 | `App.Handler() http.Handler` exposes the assembled app for tests and embedding. | Must |
| FRK-SRV-021 | `App.Run(addr string) error` blocks, serving over HTTP and shutting down gracefully on SIGINT/SIGTERM. | Must |
| FRK-SRV-022 | Graceful shutdown drains in-flight requests with a bounded timeout; forced-exit after timeout. | Must |
| FRK-SRV-023 | `App.Run` logs startup (`url`) and shutdown completion via `slog`. | Should |

### 5.4 Assets & Mono-binary

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-SRV-030 | Component CSS (`ui.ComponentsCSS`), theme CSS, and site style.css are collected and served/shipped the same way as SSG. | Must |
| FRK-SRV-031 | Assets mounted via `App.Static` resolve under their declared prefix without directory traversal (secure path joining). | Must |
| FRK-SRV-032 | A project can embed `site/` output and its own `public/` via `//go:embed` into one binary using `App.Static`. | Should |

### 5.5 Client Readiness (Future JS/TS Seam)

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-SRV-040 | SSR output must be JS-framework-agnostic: stable `data-ui-component` scope attributes, deterministic markup, no framework-specific attributes by default. | Must |
| FRK-SRV-041 | `PageSpec` may enable props serialization (`data-props` JSON) for client mounting, off by default and JSON-compatible with the API contract (KWF-230KF). | Must |
| FRK-SRV-042 | The page render path exposes an empty-by-default script/mount injection slot so a future client layer can attach JS/TS bundles without altering server rendering. | Should |
| FRK-SRV-043 | Client bundles shipped by a project (via `App.Static`) are served untouched; the theme-toggle stays the only inline script by default and coexists with mounted frameworks. | Must |

## 6. Non-Functional Requirements

- NFR1 — **Memory safety.** The `unsafe` package must not be used.
- NFR2 — **Dependencies.** `framework/web` uses stdlib + `libs` only.
- NFR3 — **Performance.** Dynamic render does not recompile templates per request (parse once, execute per request). Static export stays deterministic.
- NFR4 — **Portability.** Linux, macOS, Windows.
- NFR5 — **Quality gates.** `gofmt`, `go vet ./...`, `go test ./...` pass.

## 7. Success Criteria

- S1 — An app serves a dynamic page whose `Data` reflects a query param, and JSON API responses from the same port.
- S2 — `App.Export` output for a static page byte-matches the same page rendered through KWF-PT8OD.
- S3 — SIGINT shuts the server down gracefully (in-flight request completes).
- S4 — A single binary embeds `site/` and serves it without external files.
- S5 — The same `ui` component renders identically served live and exported.

## 8. Related Specifications

| SpecID      | KWF-0F2EB                              |
| --------- | ----------------------------------------------- |
| [KWF-M07QS](./KWF-WEB-M07QS-krewire-web-framework.md) | Krewire Web Framework (host) |
| [KWF-230KF](./KWF-HTTP-230KF-http-api-pipeline.md) | HTTP & API Pipeline |
| [KWF-C4087](./KWF-APP-C4087-krewire-app-framework.md) | Krewire App Framework (assembly) |
| [KWF-PT8OD](./KWF-SSG-PT8OD-static-site-generator.md) | Static Site Generator (shared render model) |
| [KWF-PPUWX](./KWF-UI-PPUWX-layout-ui-system.md) | Layout & UI System (registry with `Default()`) |
| [KWF-F2TQC](./KWF-JS-F2TQC-js-ts-framework-integration.md) | JS/TS Framework Integration (future bridge) |
| [KWN-6K41E](https://github.com/krewire/krewire/blob/main/docs/specs/KWN-RUN-6K41E-krewire-run-dev-deploy.md) | krewire run/dev/deploy |

## 9. References

- [KWF-M07QS](./KWF-WEB-M07QS-krewire-web-framework.md) — Krewire Web Framework.
- Go stdlib `net/http`, `os/signal`, `context`.