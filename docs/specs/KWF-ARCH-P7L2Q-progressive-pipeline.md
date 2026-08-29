# Specification — Progressive Pipeline — Static → Monolith → Services → Mesh

| Field  | Value                                              |
| ------ | -------------------------------------------------- |
| SpecID | KWF-ARCH-P7L2Q                                     |
| Title  | Progressive Pipeline — Static → Monolith → Services → Mesh |
| Status | Draft                                              |
| Date   | 2026-08-26                                         |
| Author | Krewire Contributors                                |
| Domain | Framework — Architecture / Progressive Framework   |

## 1. Context

Krewire's core promise (*KWF-M8K2Q*, `internal/docs/philosophy.md:7`) is **one Go framework for every web-service workload** with a single `krewire.yaml` and one CLI (`kiw`). Until now the *how* products grow inside that frame was scattered:

- `KWF-PT8OD` / `mdbind` give a static landing/book with zero server cost,
- `KWF-T4X9P` adds Go→WASM islands for interactivity,
- `KWF-C4087` + `KWF-5ZHQV` define the fullstack monolith and its modular (layered) variant,
- `KWF-B7N3D` owns the plan/apply infra track,
- `KWF-L5H2F` owns `worker` + `service` (registry/gateway/resilience/tracing/messaging).

No single document states the **contract of progression**: which stages exist, what guarantees hold between them, and that each stage is opt-in/zero-cost/reversible. Teams therefore copy the naïve linear pipeline

```
static (SSG/book) → layered monolith → frontend/backend split → microservice → mesh
```

and ask: is this correct and what is missing?

This spec answers that and makes the pipeline **typable** via `libs/core.Scope` (`Workspace→Module→Domain→Service→Unit`) so tooling, scaffolds and `kiw build`/`deploy` can enforce it. The ceiling for now is **Mesh (P6)** — no Platform/Federated beyond it.

## 2. Problem Statement

- **No progressive contract.** A student landing page, a campus SaaS, and a marketplace share the same `krewire new` today; there is no documented ladder of *pay only for what you need* and no guard against premature distribution.
- **Incomplete linear model.** The folk pipeline above omits two load-bearing parallel tracks — **workers** (queues/cron/DLQ) and **infra** (provider/state/plan) — both of which can appear *before* the first service extraction. It also treats `frontend/backend split` as a stage that necessarily follows a monolith, when in Krewire the WASM runtime is an *island upgrade* of SSG, not a rewrite. Data decomposition and `cli` tooling are missing entirely.
- **Ambiguous terminology.** “Frontend/backend terpisah” means different things for Astro (JS) vs Krewire (Go/WASM + Go API); without a definition teams rebuild in JS.

Cost of leaving this unsolved: rewrite tax at each growth spurt, vendor lock-in at the edges, and students taught the *wrong* mental model of how Indonesian products affordably scale.

## 3. Goals

- G1 — Define the **canonical progressive pipeline up to Mesh** with named stages, entry/exit criteria, and the `krewire.yaml`/`kind`/`package` mapping for each.
- G2 — **Critique and correct** the folk linear pipeline: show what is right, what is missing, and when branching/parallelism is valid.
- G3 — State **transition guarantees** (opt-in, zero-cost when unused, reversible by provider swap, no rewrite) per `KWF-5ZHQV` so every stage earns its complexity.
- G4 — Drive philosophy updates in `internal/docs/philosophy.md:7`, `framework/docs/philosophy.md`, `README.md`, `internal/docs/project-vision.md`, `internal/docs/architecture.md`, and `libs/docs/philosophy.md`.

## 4. Non-Goals

- NG1 — Not prescribing business-specific domain decomposition (DDD tactics remain team choice; only the *container* shape `internal/<domain>/{domain,impl,http}` from `KWF-5ZHQV` is normative).
- NG2 — Not choosing cloud vendors beyond the first two (AWS + Kubernetes per `KWF-B7N3D`); the mesh layer is provider-agnostic.
- NG3 — Not adding sidecar data-plane proxies in v1 (`KWF-L5H2F` NG1); mesh here is **library mesh**, with a sidecar slot for later.
- NG4 — Not rewriting `KWF-M8K2Q`; this spec *extends* it with a named pipeline.
- NG5 — Not covering Platform/Federated (cells, data mesh, IDP, planetary) — intentionally out of scope until Mesh is shipped and proven.

## 5. Requirements

### 5.1 Progressive Philosophy

| ID          | Requirement | Scope | Priority |
|-------------|-------------|-------|----------|
| FRK-PRG-001 | Krewire is a **progressive framework**: a product may be useful at the first stage and never leave it; every later stage adds one typed battery without invalidating artifacts or mental models from earlier stages. | Workspace | Must |
| FRK-PRG-002 | Each stage is **opt-in and zero-cost when unused** (`KWF-M8K2Q` NFR): importing only `framework/app` without `runtime`/`worker`/`service`/`infra` leaves `go vet`/`go test` and binary size indistinguishable from the monolith baseline. | Module | Must |
| FRK-PRG-003 | Each stage is **reversible** via a one-line provider swap per `KWF-5ZHQV` extraction: `impl` ↔ service client, local queue ↔ NATS, local file state ↔ S3/DynamoDB — no rewrite, contract tests stay green. | Domain | Must |
| FRK-PRG-004 | `krewire.yaml` `project.kind` remains the single dispatch key; `kiw build`/`serve`/`run`/`dev`/`worker`/`deploy`/`generate` behavior is defined per stage (see 5.4). | Module | Must |
| FRK-PRG-005 | All progression must respect `libs/core.Scope` (`Workspace ⊃ Module ⊃ Domain ⊃ Service ⊃ Unit`, `KWL-ARCH-J2K9Q`): Workspace = hub, Module = repo, Domain = bounded context, Service = deployable (`package main`), Unit = smallest code unit. | Module | Must |

### 5.2 Canonical Pipeline — Corrected & Completed (P0→P6)

```
P0  STATIC                  site | book                      framework/web/ssg + mdbind + ui
P1  INTERACTIVE STATIC      SSG + islands (WASM)            + runtime (KWF-T4X9P, hydration)
P2  MONOLITH                fullstack app (SSR+API+DB)      + app (KWF-C4087)
P3  MODULAR MONOLITH        layered monolith (domains)      + 5ZHQV structure (internal/<d>/domain|impl|http)
P4  HEADLESS SPLIT          frontend ↔ backend (optional)   runtime split (BFF) + web/api contracts
P5  DISTRIBUTED             services + workers + infra      + service/worker/infra (KWF-L5H2F + KWF-B7N3D)
P6  MESH                    library mesh (ceiling)          registry/gateway/resilience/tracing/messaging
```

`Mesh is the ceiling for now.` Fork `P0→P2` and `P5` fan-out are valid variants (see notes).

| ID          | Requirement | Scope | Priority |
|-------------|-------------|-------|----------|
| FRK-PRG-010 | **P0 — Static** (`site`/`book`): declarative `ssg:` + `content/` + `ui.Theme`; output `.krewire/build` → CDN/S3; `kiw build` → static artifacts only; zero server runtime; the cheapest stage and the correct start for landing, docs, and validation before code. | Module | Must |
| FRK-PRG-011 | **P1 — Interactive static**: `P0` plus `runtime` islands. SSG emits `data-kiw-island` markers; `GOOS=js GOARCH=wasm` build hydrates with `hydrate="load|idle|visible"`; SSR HTML remains complete without JS. This is an *upgrade*, not a rewrite, of P0. | Module | Must |
| FRK-PRG-012 | **P2 — Monolith** (`app`): `tui`+`web`+`ui`+`app` in one binary, single DB, in-process calls; `internal/app` is the sole composition root; `kiw run`/`dev` run the binary. This is Krewire's **default for SaaS** after product-market fit, not P0 — use P0 for pre-fit content. | Module | Must |
| FRK-PRG-013 | **P3 — Modular monolith** (layered): `P2` plus `KWF-5ZHQV` structure (`internal/<domain>/{domain,impl,http}` + `shared/`); contracts (`domain`) are importable; `impl` never imported outside its module + root; cross-module calls via injected interfaces or in-process `EventBus`. This stage is **where most teams should stay**; it supports extraction but does not require it. | Domain | Must |
| FRK-PRG-014 | **P4 — Headless split** (optional, not mandatory): frontend artifact (SSG or WASM) deploys to edge/CDN, backend API deployable to compute; shared contracts (`domain` DTOs) versioned; auth via JWT/cookie + CORS/CSRF (`KWF-WEB-*`); the split is **Go→Go** (WASM frontend + Go API), not Go↔JS, preserving one language. Teams that do not need separate deploy cadence *skip* P4. | Service | Should |
| FRK-PRG-015 | **P5 — Distributed** (`P3` → parallel fan-out): (a) **Services** extracted per checklist (P3 `domain` → shared contracts module, `impl` → `service/<name>` main, DB per service or schema-per-service), (b) **Workers** (`worker` queues/cron/DLQ, `KWF-L5H2F` §5.7) for async work, (c) **Infra** (typed resources + `Plan` pure + state/locking, `KWF-B7N3D`). Any subset of (a)(b)(c) may be adopted; they are **peer batteries**, not sequential sub-stages. | Service | Must |
| FRK-PRG-016 | **P6 — Mesh** (requires P5, **ceiling**): **library mesh** per `KWF-L5H2F`: `service/registry` (Consul/etcd/NATS/DNS) + `service/config` + `web/gateway` + `service/resilience` (circuit/bulkhead/retry) + `service/tracing` (OTel/W3C) + `service/messaging` (NATS JetStream). Must remain opt-in; monolith without it stays zero-cost. | Service | Must |

**Critique of the folk pipeline — what is right vs missing:**

| Folk element | Verdict | Detail |
|--------------|---------|--------|
| `static → layered/modular monolith` | **Correct** | Folk is right to start cheap and introduce boundaries early (`KWF-5ZHQV`). Rename `layered` → `modular monolith` for a canonical import path `internal/<domain>` to avoid “n-tier” confusion. |
| `frontend/backend split` | **Correct but optional** | In Krewire this is a *deployment* split (edge WASM vs API), not a language split; many teams should stay on P3 integrated SSR instead. Do not force P4 before P5. |
| `→ microservice → mesh` | **Correct order, missing parallelism** | Folk omits **workers** and **infra** as peer batteries of microservices. Real pipeline is `P3 → {services, workers, infra} in parallel → mesh synthesizes them`. A job queue (`worker`) often precedes the first service extraction. |
| Missing | **Data layer** | Folk has no DB decomposition step: shared DB → schema-per-domain → DB-per-service. Make it visible at P5 before claiming “service done.” |
| Missing | **CLI** | `cli` kind (`framework/tui`) is a *leaf* not a pipeline stage; internal tooling (`kiw generate`, ops CLIs) may be scaffolded at any stage and never advances the pipeline. |
| Missing | **Observability** | Tracing/logs/metrics are cross-cutting (gateway + tracing), not a stage; they appear at P5 and mature at P6 — document as such. |

### 5.3 Transition Guarantees

| ID          | Requirement | Scope | Priority |
|-------------|-------------|-------|----------|
| FRK-PRG-020 | Every stage's artifacts are **additive**: SSG output preserved when `runtime` is added (hydration parity per `KWF-T4X9P` NFR1); monolith routes preserved when modularized; service contracts are Go interfaces, not new wire protocols. | Unit | Must |
| FRK-PRG-021 | **Scaffold path** (`kiw new` → kernel → `kiw init [--site|--book|--cli]`; worker/service/infra planned) must emit the correct layout for the target stage and pass `gofmt`/`go vet`/`go test` with the minimal imports of that stage. | Module | Must |
| FRK-PRG-022 | **Deployment dispatch** per stage: P0/P1 `kiw build` → site; P2/P3 `kiw run`/`kiw dev`; P5 `kiw worker` + `kiw deploy --plan/--preview`; P6 `kiw dashboard`. No stage introduces a second build script or config file beyond `krewire.yaml`. | Workspace | Must |
| FRK-PRG-023 | **State discipline**: P0–P3 use local `.krewire/` state; remote state/locking (S3/DynamoDB, Consul) appears only at P5/P6 (`KWF-B7N3D` FRK-INFRA-020/021) and must degrade gracefully when the new battery is imported. | Service | Must |

### 5.4 Kind & Package Map (normative)

| Stage | `krewire.yaml` `project.kind` | New import added | `kiw` command that becomes live |
|-------|------------------------------|------------------|----------------------------------|
| P0 | `site` or `book` | `framework/web/ssg` or `mdbind`, `framework/ui` | `kiw build`, `kiw serve` |
| P1 | `site`/`book` + `runtime` opt-in | `framework/runtime`, `framework/ui` Theme | `kiw build` now also builds WASM |
| P2 | `app` | `framework/web`, `framework/app`, `framework/dsl` | `kiw run`, `kiw dev` |
| P3 | `app` (structured) | `KWF-5ZHQV` conventions (`internal/<d>`) | same; `krewire verify modules` (future) |
| P4 | `app` + `site` (two deploys) | `framework/runtime` split + CORS/CSRF | `kiw build` (frontend) + `kiw run` (API) |
| P5 | `service`, `worker`, `infra` (any subset) | `framework/service`, `framework/worker`, `framework/infra` | `kiw worker`, `kiw deploy`, `kiw deploy --preview` |
| P6 | `service`/`infra` (ceiling) | `service/registry|config|gateway|resilience|tracing|messaging` | `kiw dashboard` |

## 6. Non-Functional Requirements

- NFR1 — **Library-first, stdlib-first** (`KWF-M8K2Q` NFR1): new stages add deps only when stdlib lacks the primitive (OTel, NATS); a bare `app` stays import-clean.
- NFR2 — **Zero-cost proof**: `gofmt -l .`, `go vet ./...`, `go test ./...` remain green in every repo at every stage; scaffold fixtures for each stage compile and run the pipeline's `kiw` command.
- NFR3 — **Idiomatic Go**: `(value, error)` signatures, functional options, zero-value usability, `go doc` examples for every new public type.
- NFR4 — **Docs as code**: `.krewire/build` output for P0/P1 fixtures is `curl`-readable without JS; `kiw build --plan` for `infra` prints a plan without touching state (KWF-B7N3D NFR2).

## 7. Success Criteria

- S1 — A PR landing (P0) is upgraded to P1 islands, then promoted to P2 monolith, then refactored to P3 modular layout — **no file is moved twice** and every stage’s `go test ./...` passes; contract test for one module stays green after `impl` → service client swap at P5.
- S2 — `framework/docs/architecture.md` and `internal/docs/architecture.md` each contain a “Progressive Pipeline” diagram with P0→P6 and the kind/package table (5.4) plus the critique table (5.2).
- S3 — `krewire info` and `AGENTS.md` Progressive note correctly identify 8 kinds and map them to P0→P6; a new contributor can follow the pipeline from site to mesh without asking.
- S4 — `krewire deploy --preview` provisions a `pr-*` ephemeral env in CI against the local/file backend and tears down (`--destroy`) without manual state edits.

## 8. Related Specifications

| SpecID | Title |
|--------|-------|
| [KWF-ARCH-M8K2Q](./KWF-ARCH-M8K2Q-unified-framework-vision.md) | Unified Framework Vision (parent) |
| [KWF-ARCH-5ZHQV](./KWF-ARCH-5ZHQV-modular-monolith-architecture.md) | Modular Monolith & Extraction Path |
| [KWF-WASM-T4X9P](./KWF-WASM-T4X9P-wasm-client-runtime.md) | WASM Client Runtime (P1) |
| [KWF-INFRA-B7N3D](./KWF-INFRA-B7N3D-cloud-provider-abstraction.md) | Cloud Provider Abstraction (P5 infra) |
| [KWF-SVC-L5H2F](./KWF-SVC-L5H2F-microservice-patterns.md) | Service & Worker Patterns (P5/P6 mesh) |
| [KWL-ARCH-J2K9Q](https://github.com/krewire/libs/blob/main/docs/specs/KWL-ARCH-J2K9Q-ecosystem-scope-levels.md) | Ecosystem Scope Levels (Scope vocabulary) |
| [KWF-STRUCT-CCI0N](./KWF-STRUCT-CCI0N-app-directory-structure.md) | App Directory Structure |

## 9. References

- Sam Newman — *Monolith to Microservices* (2019): incremental extraction, not big-bang.
- CNCF — *Service Mesh* patterns: library mesh vs sidecar mesh (Krewire chooses library first per `KWF-L5H2F` NG1).
- PWA / Islands architecture — Astro: https://docs.astro.build/en/concepts/islands/
- Go WebAssembly: https://go.dev/wiki/WebAssembly
- OpenTelemetry / NATS JetStream — see `KWF-L5H2F` refs.

