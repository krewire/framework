# Specification — Krewire App Framework

| Field       | Value                                       |
| ----------- | ------------------------------------------- |
| SpecID      | KWF-C4087                              |
| Title       | Krewire App Framework — Monolith Assembly    |
| Status      | Draft                                       |
| Date        | 2026-08-19                                  |
| Author      | Krewire Contributors                         |
| Domain      | Frameworks — App / Monolith                 |

## 1. Context

The full stack is now addressable: static sites (KWF-PT8OD), JSON APIs
(KWF-230KF), and server-rendered frontends (KWF-0F2EB). **Monolith Assembly**
defines how a *project* becomes one runnable application: the project's
`main.go` composes `web.App` from routing, middleware, pages, static assets, and
configuration, then ships as a single binary that serves HTML + API + assets.

This spec is the contract between `framework` and the projects that assemble
apps. It supersedes the "no per-project `cmd`" convention **for app projects
only**: static/book sites keep the config-driven `krewire build` path, while
fullstack apps own an explicit `main.go` so backend logic lives in Go, is
`go run`-able, and has no generated or hidden code.

## 2. Problem Statement

Krewire can build sites but not *applications*. A fullstack monolith needs:

- Backend logic (handlers, middleware, DB wiring) expressed in Go — which the
  config-only model cannot host.
- Server-rendered pages and JSON APIs in one binary.
- Production-ready lifecycle beyond `http.ListenAndServe`.
- A single, documented assembly point so `krewire run`/`dev` (KWN-6K41E) can
  drive any project predictably.

Existing conventions were built for static delivery; extending them with a
small, explicit `main.go` per app keeps projects idiomatic without reintroducing
the "magic main" that other fullstack Go frameworks were criticized for
(KWF-PPUWX §1.1).

## 3. Goals

- G1 — Define the app-project contract: an explicit, idiomatic `main.go` that builds a `web.App`.
- G2 — Keep static/book sites entirely config-driven (unchanged convention); exempt only app projects from the no-`cmd` rule.
- G3 — Make app assembly declarative enough that `krewire run`/`dev` can predict project shape and lifecycle.
- G4 — Support configuration via `libs/config` + `libs/validate` with env overrides.
- G5 — Support single-binary deployment with embedded assets, static export, and API in one artifact.
- G6 — Keep the framework dependency direction: projects → framework → libs; never reverse.

## 4. Non-Goals

- NG1 — Generating `main.go` or any build-time code (NG6 of KWF-PPUWX still holds).
- NG2 — A plugin/hot-swap system for app components.
- NG3 — Microservices orchestration; this is single-process monolith.
- NG4 — Storage/ORM coupling: DB choice stays a project decision.
- NG5 — Changing behavior of the existing static/book `krewire build` path.

## 5. Requirements

### 5.1 App Project Contract

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-APP-001 | An app project ships an explicit `main.go` (package `main`) that builds and runs its `web.App`. | Must |
| FRK-APP-002 | App projects may also live under `cmd/<name>`, keeping `main` thin and logic in packages. | Should |
| FRK-APP-003 | The project shape is detectable: a `main.go`/`cmd/*` with `web.App` marks an *app*; `krewire.yaml`+`ssg:` marks a *site*; `manuscript/` marks a *book*. | Must |
| FRK-APP-004 | No generated code, no injected `main`, no reflection-based app assembly. | Must |

### 5.2 Assembly & Configuration

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-APP-010 | Provide `web.NewApp()` as the sole assembly entry; middleware/API via KWF-230KF, pages via KWF-0F2EB. | Must |
| FRK-APP-011 | App configuration loads through `libs/config` (KWL-2X1QZ) and validates through `libs/validate` (KWL-LHANF) before use. | Must |
| FRK-APP-012 | Config precedence is env > file > zero (as specified in KWL-2X1QZ §5.3). | Must |
| FRK-APP-013 | A startup configuration/validation error exits non-zero with a clear message (mapped through `core` exit codes). | Must |

### 5.3 Rendering & Assets

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-APP-020 | App pages reuse the `ui` registry (KWF-PPUWX): built-in components out of the box, custom components registerable. | Must |
| FRK-APP-021 | Apps may embed the SSG-exported `site/` (built via KWF-PT8OD) and serve it with `App.Static`, keeping docs/marketing pages static. | Should |
| FRK-APP-022 | All frontend rendering flows through the unified SSR render path (KWF-0F2EB FRK-SRV-014). | Must |
| FRK-APP-050 | Apps may ship compiled client bundles (JS/TS output) under `public/` or `site/` and serve them via `App.Static`; the framework never compiles, transforms, or blocks them. | Must |
| FRK-APP-051 | The default monolith remains zero-JS; a client integration is an opt-in, additive layer (KWF-F2TQC) that never alters server rendering or the theme-toggle default. | Must |
| FRK-APP-052 | `examples/app` must demonstrate serving a static `public/` with a `<script>` mount point alongside server-rendered pages, kept byte-identical without the script. | Should |

### 5.4 Lifecycle & Deployment

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-APP-030 | `main.go` calls `App.Run(addr)` for graceful start/shutdown (KWF-0F2EB FRK-SRV-021). | Must |
| FRK-APP-031 | The deliverable is one binary: embedded assets + API + SSR on one listener. | Must |
| FRK-APP-032 | Every exported identifier composing apps has a doc comment (rendered by `go doc`). | Must |

### 5.5 Org & Testing

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-APP-040 | Provide `examples/app` demonstrating a monolith: config, DTO + validation, middleware, dynamic + static pages, embedded assets. | Must |
| FRK-APP-041 | Quality gates (`gofmt`, `go vet ./...`, `go test ./...`) pass for `examples/app`. | Must |

## 6. Non-Functional Requirements

- NFR1 — **Memory safety.** The `unsafe` package must not be used.
- NFR2 — **Dependencies.** Framework uses stdlib + `libs`; apps add only what their backend needs (DB drivers, etc.).
- NFR3 — **Portability.** Linux, macOS, Windows.
- NFR4 — **Quality gates.** `gofmt`, `go vet ./...`, `go test ./...` pass.
- NFR5 — **Determinism.** Identical config/env yield identical app startup.

## 7. Success Criteria

- S1 — `examples/app` starts from `go run .`, serves a dynamic page, a JSON endpoint, and an embedded static page on one port.
- S2 — `krewire identify` (KWN-6K41E) classifies the example as an *app* automatically.
- S3 — Rebuilding `site/` with KWF-PT8OD and embedding it does not change app startup.
- S4 — A misconfigured app exits non-zero with a before-first-request error message.

## 8. Related Specifications

| SpecID      | KWF-C4087                              |
| --------- | ----------------------------------------------- |
| [KWF-M07QS](./KWF-WEB-M07QS-krewire-web-framework.md) | Krewire Web Framework (host) |
| [KWF-230KF](./KWF-HTTP-230KF-http-api-pipeline.md) | HTTP & API Pipeline |
| [KWF-0F2EB](./KWF-WEB-0F2EB-server-frontend-pipeline.md) | Server & Frontend Rendering Pipeline |
| [KWF-PT8OD](./KWF-SSG-PT8OD-static-site-generator.md) | Static Site Generator (site/ export) |
| [KWF-PPUWX](./KWF-UI-PPUWX-layout-ui-system.md) | Layout & UI System |
| [KWF-F2TQC](./KWF-JS-F2TQC-js-ts-framework-integration.md) | JS/TS Framework Integration (client asset seam) |
| [KWN-6K41E](https://github.com/krewire/krewire/blob/main/docs/specs/KWN-RUN-6K41E-krewire-run-dev-deploy.md) | krewire run/dev/deploy |
| [KWL-2X1QZ](https://github.com/krewire/libs/blob/main/docs/specs/KWL-CONFIG-2X1QZ-configuration-loading.md) | Configuration Loading |
| [KWL-LHANF](https://github.com/krewire/libs/blob/main/docs/specs/KWL-VALIDATE-LHANF-struct-validation.md) | Struct Validation |

## 9. References

- [KWF-CMBZJ](./KWF-META-CMBZJ-krewire-meta-framework.md) — Krewire Meta-Framework (module architecture).
- Go stdlib `net/http`, `os/signal`.