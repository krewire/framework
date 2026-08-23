# Specification — Unified Framework Vision

| Field  | Value                              |
| ------ | ---------------------------------- |
| SpecID | KWF-M8K2Q                          |
| Title  | Unified Framework Vision — One Framework for Every Web Service Workload |
| Status | Draft                              |
| Date   | 2026-08-21                         |
| Author | Krewire Contributors                |
| Domain | Framework — Architecture           |

## 1. Context

Krewire began as a meta-framework composing modules across ecosystems into one
cohesive Go foundation (KWF-CMBZJ). Today it ships five workload shapes —
`app`, `cli`, `site`, `book`, and the pre-`init` kernel — driven by a single
CLI (`krewire`) and a single config file (`krewire.yaml`). The web stack covers
SSR, JSON APIs, a declarative SSG (KWF-PT8OD), theming and scoped CSS
(KWF-0Z671), and a modular-monolith default with an extraction path
(KWF-5ZHQV).

Product teams, however, still reach for foreign toolchains to cover the rest
of the spectrum: Node CLIs and bundlers for frontend interactivity, Terraform
or point-and-click consoles for infrastructure, Spring-ish stacks for service
discovery and resilience, Celery/Sidekiq for background work. Each addition
reintroduces the fragmentation Krewire exists to remove.

## 2. Problem Statement

- Building a product that spans CLI + site + API + workers + infra requires
  **four or more toolchains**, config formats, and deploy stories.
- Client-side interactivity currently requires a JS/TS bridge (KWF-F2TQC),
  splitting the component model and toolchain by language.
- There is no Go-native path from "declare infrastructure" to "deployed", so
  `krewire deploy` cannot own the full lifecycle.
- Extracting a module (KWF-5ZHQV checklist) lands teams in an ecosystem with
  no discovery, gateway, resilience, or tracing support.
- Background jobs have no first-class framework: no queues, cron, retries,
  or dead-letter handling.

## 3. Goals

- G1 — Make Krewire a credible single framework for **ten workloads**: CLI,
  worker, cloud infra, SSG, doc site, frontend, backend, fullstack, monolith,
  microservice.
- G2 — Add a **Go-native client runtime** compiled with the standard Go
  toolchain (`GOOS=js GOARCH=wasm`) — no TinyGo, no JS bridge for v1.
- G3 — Add a **multi-cloud provider abstraction** (library-first IaC) with
  AWS and Kubernetes as first implementations.
- G4 — Add **opt-in microservice and worker patterns** aligned with the
  modular-monolith extraction path.
- G5 — Extend the **unified CLI** (`krewire deploy`, `krewire dashboard`,
  `krewire generate`) so every kind shares one command matrix.
- G6 — Deliver in phases, each gated by an end-to-end demo and its own spec.

## 4. Non-Goals

- NG1 — Not replacing the Go toolchain or building a custom compiler; Krewire
  builds on standard `go build` / `GOOS=js GOARCH=wasm`.
- NG2 — Not a lock-in PaaS: provider abstractions must keep escape hatches
  (raw resource passthrough, terraform-compatible state export later).
- NG3 — Not porting Flutter or Astro wholesale — re-implementing their
  *capabilities* idiomatically in Go, scoped to what web products need.
- NG4 — No committed `go.work`; cross-repo testing uses temporary `replace`
  directives per existing policy.
- NG5 — Not forcing distributed patterns on monoliths: everything beyond
  `app` is opt-in via project kind or explicit imports.

## 5. Requirements

### 5.1 Workload Matrix & Project Kinds

| ID           | Requirement                                                              | Priority |
| ------------ | ------------------------------------------------------------------------ | -------- |
| FRK-UNI-001  | `project.kind` accepts `worker`, `service`, and `infra` in addition to    | Must     |
|              | `app`, `cli`, `site`, `book`. Detection extends `krewire info`.            |          |
| FRK-UNI-002  | Each kind maps to a defined package set and command subset (see 5.4).     | Must     |
| FRK-UNI-003  | A single repository may declare exactly one primary kind; composite       | Should   |
|              | products use a monolith `app` with modules (KWF-5ZHQV).                   |          |
| FRK-UNI-004  | Unknown kinds fail with usage error exit code (2), never silently build.  | Must     |

### 5.2 Configuration

| ID           | Requirement                                                              | Priority |
| ------------ | ------------------------------------------------------------------------ | -------- |
| FRK-UNI-010  | All configuration lives in `krewire.yaml`; typed structs loaded via        | Must     |
|              | `libs/config`, validated via `libs/validate` tags.                        |          |
| FRK-UNI-011  | Kind-specific sections (`worker:`, `service:`, `infra:`) are optional     | Must     |
|              | and validated only when the matching kind is selected.                    |          |
| FRK-UNI-012  | Secrets are referenced, never stored: `${env:NAME}` or secrets-manager    | Must     |
|              | URIs resolved at runtime.                                                 |          |

### 5.3 Package Architecture

Target layout inside `github.com/krewire/framework`:

```
framework/
├── tui/       # shipped
├── web/       # shipped (+ ssg/)
├── ui/        # shipped
├── app/       # shipped
├── runtime/   # client runtime: js bridge, vdom, components, widgets, hydration
├── worker/    # job queues, cron, retries, DLQ
├── service/   # registry, config center, gateway, resilience, tracing, messaging
└── infra/     # provider contract, schema, state+lock, plan/apply, aws/, k8s/
```

| ID           | Requirement                                                              | Priority |
| ------------ | ------------------------------------------------------------------------ | -------- |
| FRK-UNI-020  | New packages are separate import paths; importing `runtime` must not      | Must     |
|              | pull `infra`, and vice versa (no cross-imports between new packages).     |          |
| FRK-UNI-021  | Every new package compiles for both host GOOS and `GOOS=js GOARCH=wasm`   | Must     |
|              | where it participates in client rendering.                                |          |
| FRK-UNI-022  | Zero-cost when unused: an `app` that imports none of the new packages     | Must     |
|              | produces a binary unchanged in size and behavior.                         |          |

### 5.4 Unified CLI Command Matrix

| Command            | app/cli | site/book | worker | service | infra |
| ------------------ | ------- | --------- | ------ | ------- | ----- |
| `krewire run/dev`   | ✅      | —         | ✅     | ✅      | —     |
| `krewire build`     | binary  | site/     | binary | binary  | plan  |
| `krewire deploy`    | ✅      | ✅ static | ✅     | ✅      | apply |
| `krewire test`      | ✅      | ✅        | ✅     | ✅      | ✅    |
| `krewire dashboard` | —       | —         | ✅     | ✅      | ✅    |
| `krewire generate`  | ✅      | ✅        | ✅     | ✅      | ✅    |

| ID           | Requirement                                                              | Priority |
| ------------ | ------------------------------------------------------------------------ | -------- |
| FRK-UNI-030  | `krewire deploy` detects kind → builds artifacts → provisions/applies →    | Must     |
|              | reports URLs/endpoints; supports `--plan`, `--auto-approve`, `--destroy`. |          |
| FRK-UNI-031  | `krewire dashboard` serves a local UI aggregating services, logs, traces,  | Should   |
|              | job queues, and infra state for kinds that produce runtime processes.     |          |
| FRK-UNI-032  | `krewire generate` subcommands (`openapi`, `config`) emit compiling Go     | Should   |
|              | code; generators are additive and idempotent.                             |          |

### 5.5 Phased Delivery

| ID           | Requirement                                                              | Priority |
| ------------ | ------------------------------------------------------------------------ | -------- |
| FRK-UNI-040  | Phase 1 (client runtime) is specified by KWF-T4X9P and demos before       | Must     |
|              | Phase 2 starts.                                                           |          |
| FRK-UNI-041  | Phase 2 (cloud infra) is specified by KWF-B7N3D.                          | Must     |
| FRK-UNI-042  | Phase 3 (microservice + worker patterns) is specified by KWF-L5H2F.       | Must     |
| FRK-UNI-043  | Each phase ends with: green gates in touched repos, docs regenerated,     | Must     |
|              | new tags propagated to downstream `go.mod` files per release policy.      |          |

## 6. Non-Functional Requirements

- NFR1 — **Stdlib-first** persists: new packages add third-party dependencies
  only when the stdlib genuinely lacks the capability (e.g., OTel SDK).
- NFR2 — **Quality gates** (`gofmt -l .`, `go vet ./...`, `go test ./...`)
  pass in every touched repo before any phase is declared done.
- NFR3 — **Idiomatic Go APIs**: `(value, error)` signatures, functional
  options where suitable, zero-value usability, clear go doc comments.
- NFR4 — **No committed `go.work`**; cross-repo work uses temporary local
  `replace` directives removed before commits.

## 7. Success Criteria

- S1 — `krewire info` correctly reports all seven kinds on fixture projects.
- S2 — A demo product (site + API + worker + infra declaration) builds and
  deploys end-to-end using only `krewire` commands and `krewire.yaml`.
- S3 — An `app` project that uses none of the new packages shows no binary
  size or behavior change versus current v0.5.x output.
- S4 — Each phase's child spec lists its own acceptance criteria and they are
  demonstrably met before the next phase begins.

## 8. Related Specifications

| SpecID    | Title                                                        |
| --------- | ------------------------------------------------------------ |
| [KWF-CMBZJ](./KWF-META-CMBZJ-krewire-meta-framework.md)   | Meta-framework foundation |
| [KWF-C4087](./KWF-APP-C4087-krewire-app-framework.md)     | App framework assembly    |
| [KWF-PT8OD](./KWF-SSG-PT8OD-static-site-generator.md)    | Static site generator     |
| [KWF-0Z671](./KWF-UI-0Z671-krewire-ui-framework.md)       | UI framework              |
| [KWF-5ZHQV](./KWF-ARCH-5ZHQV-modular-monolith-architecture.md) | Modular monolith & extraction path |
| [KWF-T4X9P](./KWF-WASM-T4X9P-wasm-client-runtime.md)     | WASM client runtime (Phase 1) |
| [KWF-B7N3D](./KWF-INFRA-B7N3D-cloud-provider-abstraction.md) | Cloud provider abstraction (Phase 2) |
| [KWF-L5H2F](./KWF-SVC-L5H2F-microservice-patterns.md)    | Microservice & worker patterns (Phase 3) |

## 9. References

- Astro islands architecture: https://docs.astro.build/en/concepts/islands/
- Flutter architecture (widget/element/render trees): https://docs.flutter.dev/resources/architectural-overview
- Pulumi programming model: https://www.pulumi.com/docs/concepts/
- OpenTelemetry specification: https://opentelemetry.io/docs/specs/otel/
- Go WASM: https://go.dev/wiki/WebAssembly
