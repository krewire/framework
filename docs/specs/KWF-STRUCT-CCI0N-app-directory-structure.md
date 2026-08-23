# Specification — App Directory Structure

| Field       | Value                                       |
| ----------- | ------------------------------------------- |
| SpecID      | KWF-CCI0N                              |
| Title       | App Project Directory Structure Standard    |
| Status      | Draft                                       |
| Date        | 2026-08-19                                  |
| Author      | Krewire Contributors                         |
| Domain      | Framework — Project Structure              |

## 1. Context

Krewire applications are fullstack monoliths: one binary serves server-rendered
pages (KWF-0F2EB), JSON APIs (KWF-230KF), static sites (KWF-PT8OD), and
embedded assets composed through `web.App` (KWF-C4087). Modern fullstack
frameworks define a canonical project layout — Laravel's `app/`, `config/`,
`resources/`, Astro/SvelteKit's `src/` — so new projects start conventional and
tooling behaves predictably. Krewire defines the same kind of standard for Go,
then keeps it a *convention* rather than a hard rule.

Today `krewire new` emits only a bare CLI skeleton and earlier example projects
predate any layout standard. Without a canonical structure every app invents
its own arrangement, divergence grows, and `krewire run`/`dev`/`init`
(KWN-6K41E, KWN-RD3WS, KWN-1QGI2) lose the predictability they depend on.

## 2. Problem Statement

There is no defined, *ideal* directory structure for a Krewire app. Teams copy
idiosyncratic layouts between projects; the devtool cannot offer a
conventional default; and newcomers have no canonical answer for "where do the
templates, styles, handlers, and entry point live?". A standard that is
idiomatic Go while matching the expectations of modern frameworks closes that
gap — provided it stays a default users may reorganize, never a prison.

## 3. Goals

- G1 — Define a canonical app directory structure that is idiomatically Go and
  conventional for modern fullstack frameworks.
- G2 — Scaffold it deterministically with `krewire new` (app) and `krewire init`
  (site/book).
- G3 — Keep the structure a convention: users may relocate directories when
  they declare the move explicitly; no tool hardcodes magic paths.
- G4 — Keep every project single-binary friendly: entry point, internal
  packages, UI sources, and static assets all embed via stdlib `embed`.
- G5 — Preserve existing shape detection (app/site/book) and the static
  book/site pipeline (KWN-1QGI2) unchanged.

## 4. Non-Goals

- NG1 — Enforcing the layout across all projects; it is a recommended default
  and scaffold output, not a linter or a hard gate.
- NG2 — Codifying multi-process or microservice layouts; this standard covers
  the single-process monolith.
- NG3 — Reflection-driven or auto-discovery assembly; paths are declared, not
  guessed beyond stable markers.
- NG4 — Client framework specifics; the JS/TS bridge (KWF-F2TQC) covers how
  compiled frontends attach, not where they must live.

## 5. Requirements

### 5.1 Canonical App Layout

A `krewire new <name>` app project follows this layout by default:

```
<name>/
├── cmd/
│   └── <name>/
│       └── main.go          # thin entry: load config, build app, run
├── internal/
│   ├── app/                 # assembly: New(deps) *web.App (all wiring here)
│   ├── config/              # typed config: struct + Load() + Validate()
│   ├── http/                # handlers, routes, middleware (API + SSR)
│   └── …                    # domain/store/service packages (choice)
├── web/
│   ├── components/          # reusable components (templates/registry entries)
│   ├── layouts/             # page shells
│   ├── pages/               # SSR page definitions (data + mounts)
│   └── theme/               # palette, tokens, toggle styles
├── public/                  # static assets served as-is (embedded)
├── manuscript/              # optional book source (docs/marketing)
├── krewire.yaml              # project config at the repo root
├── go.mod
├── go.sum
├── README.md
└── .gitignore
```

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-STR-001 | App projects default to the canonical layout: `cmd/<name>`, `internal/`, `web/`, `public/`, root `krewire.yaml`. | Must |
| FRK-STR-002 | The entry point is `cmd/<name>/main.go` (a root `main.go` is allowed for single-command apps); it must stay thin — load config, build the app, call `App.Run` — never host business logic. | Must |
| FRK-STR-003 | App logic lives in `internal/` packages by default; anything importable by the outside world belongs under `pkg/` and is the project's choice. | Must |
| FRK-STR-004 | Full assembly composes in `internal/app` (e.g. `app.New(cfg, store) *web.App`); `cmd` only calls it. | Should |
| FRK-STR-005 | Configuration is a typed struct in `internal/config`, loaded through `libs/config` (KWL-2X1QZ) and validated through `libs/validate` (KWL-LHANF); the config file is `krewire.yaml` at the repo root. | Must |
| FRK-STR-006 | Frontend sources (components, layouts, pages, theme, styles) live under `web/` by default and register into the app at assembly time; built-in components come from the `ui` registry (KWF-PPUWX). | Must |
| FRK-STR-007 | Static assets served as-is live under `public/` and are embedded via `//go:embed public/...` into the binary. | Must |
| FRK-STR-008 | An optional book source is `manuscript/` (KWN-1QGI2); an optional exported SSG site is `site/`. Apps may serve either through `App.Static`. | Should |
| FRK-STR-009 | The layout is a convention, not a contract: no framework or devtool code may hardcode these paths beyond the stable markers in FRK-STR-030 through FRK-STR-032. | Must |
| FRK-STR-010 | A project may relocate any directory by declaring the move centrally in `krewire.yaml` (e.g. `project.dirs.web: src/web`); un-declared relocations are the author's responsibility. | Should |

### 5.2 Scaffolding

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-STR-020 | `krewire new <name>` scaffolds the canonical app layout (FRK-STR-001) with a runnable hello page, one JSON endpoint, and `krewire.yaml`; it builds and runs without modification. | Must |
| FRK-STR-021 | `krewire init` scaffolds the site/book layout: `manuscript/` plus `krewire.yaml` with shared fields (title, input, output, theme, optional `ssg:`). | Must |
| FRK-STR-022 | Scaffolded projects must work with `krewire run`, `krewire dev`, `krewire build`, and `krewire serve` exactly as documented (KWN-6K41E). | Must |
| FRK-STR-023 | Scaffolding keeps its safety contract: refuse non-empty targets and invalid names with exit code 2, list created files deterministically (KWN-RD3WS). | Must |
| FRK-STR-024 | Scaffolded output pins the framework/libs versions the devtool was built with (KWN-RD3WS RND-SC-006). | Must |

### 5.3 Discovery & Tooling

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-STR-030 | Shape detection (KWN-6K41E RND-SHD-001..004) stays marker-based: `main.go`/`cmd/*` with `func main` ⇒ app; `krewire.yaml#ssg:` or `ssg.yaml` ⇒ site; `manuscript/` ⇒ book. It does not probe the canonical layout. | Must |
| FRK-STR-031 | The `krewire dev` watched set derives from the canonical layout — `**/*.go`, `web/**`, `public/**`, `krewire.yaml`, `ssg.yaml`, `manuscript/**` — with a `krewire.yaml` override for relocated dirs. | Should |
| FRK-STR-032 | `krewire info` reports the detected kind and, when resolved, the effective directories for `cmd`, `internal`, `web`, `public`, and `manuscript`. | Should |
| FRK-STR-033 | Applications resolve layout directories only through the declared locations (defaults or `krewire.yaml` overrides); nothing reflects on the filesystem beyond markers and declared paths. | Must |

## 6. Non-Functional Requirements

- NFR1 — **Memory safety.** The `unsafe` package must not be used.
- NFR2 — **Dependencies.** The scaffolding lives in `krewire` (stdlib + `libs/core`); framework packages may use stdlib + `libs`.
- NFR3 — **Portability.** Linux, macOS, Windows.
- NFR4 — **Quality gates.** `gofmt`, `go vet ./...`, `go test ./...` pass in every affected repo.
- NFR5 — **Determinism.** Identical input always yields the identical layout; no time-, locale-, or filesystem-order-dependent output.
- NFR6 — **Embeddability.** The canonical layout embeds through stdlib `embed` only; no generated code, no external bundler required.

## 7. Success Criteria

- S1 — `krewire new demo` produces a project where `krewire run` serves a page and a JSON endpoint, and `krewire dev` restarts after a `web/` or `.go` edit.
- S2 — `krewire init` produces a `manuscript/` + `krewire.yaml` that `krewire build` and `krewire serve` accept unchanged.
- S3 — Declaring `project.dirs.web: src/web` in `krewire.yaml` lets `krewire dev` watch the relocated directory without code changes.
- S4 — `krewire info` reports `Project kind: app` and the effective directories.
- S5 — No existing static/book project changes behavior.

## 8. Related Specifications

| SpecID      | KWF-CCI0N                              |
| --------- | ----------------------------------------------- |
| [KWF-C4087](./KWF-APP-C4087-krewire-app-framework.md) | Krewire App Framework (assembly) |
| [KWF-C9WLJ](./KWF-DI-C9WLJ-app-container-service-providers.md) | App Container & Service Providers |
| [KWF-PPUWX](./KWF-UI-PPUWX-layout-ui-system.md) | Layout & UI System |
| [KWF-PT8OD](./KWF-SSG-PT8OD-static-site-generator.md) | Static Site Generator |
| [KWF-0F2EB](./KWF-WEB-0F2EB-server-frontend-pipeline.md) | Server & Frontend Rendering Pipeline |
| [KWF-230KF](./KWF-HTTP-230KF-http-api-pipeline.md) | HTTP & API Pipeline |
| [KWF-F2TQC](./KWF-JS-F2TQC-js-ts-framework-integration.md) | JS/TS Framework Integration (client assets) |
| [KWN-6K41E](https://github.com/krewire/krewire/blob/main/docs/specs/KWN-RUN-6K41E-krewire-run-dev-deploy.md) | krewire run/dev/deploy |
| [KWN-RD3WS](https://github.com/krewire/krewire/blob/main/docs/specs/KWN-SCAFFOLD-RD3WS-project-scaffolding.md) | Project Scaffolding |
| [KWN-1QGI2](https://github.com/krewire/krewire/blob/main/docs/specs/KWN-BUILD-1QGI2-project-building.md) | Project Building |
| [KWL-2X1QZ](https://github.com/krewire/libs/blob/main/docs/specs/KWL-CONFIG-2X1QZ-configuration-loading.md) | Configuration Loading |
| [KWL-LHANF](https://github.com/krewire/libs/blob/main/docs/specs/KWL-VALIDATE-LHANF-struct-validation.md) | Struct Validation |

## 9. References

- [KWF-CMBZJ](./KWF-META-CMBZJ-krewire-meta-framework.md) — Krewire Meta-Framework (module architecture).
- Go standard library `embed`, `internal/`, and `cmd/` project-layout idioms.