# Specification — Modular Monolith Architecture

| Field       | Value                                     |
| ----------- | ----------------------------------------- |
| SpecID      | KWF-5ZHQV                              |
| Title       | Modular Monolith Architecture Default     |
| Status      | Draft                                     |
| Date        | 2026-08-19                                |
| Author      | Krewire Contributors                       |
| Domain      | Framework — Architecture                  |

## 1. Context

Krewire applications are fullstack monoliths serving SSR pages, JSON APIs,
static sites, and embedded assets from a single binary (KWF-C4087). As projects
grow, teams often need to extract bounded contexts into separate services.
Starting with a **modular monolith** — a single deployable unit with explicit
internal module boundaries — gives the best of both worlds: simple operations
today, clean extraction paths tomorrow.

Currently the framework provides a single composition root (`internal/app`) and
an app container (KWF-C9WLJ), but does not prescribe how application code
organizes into modules. Without a standard module structure, extraction later
requires painful refactoring.

## 2. Problem Statement

- Teams build applications without clear module boundaries, coupling domain
  logic across features.
- No standard way to declare module contracts (interfaces) vs implementations.
- Extraction to microservices requires identifying and untangling implicit
  dependencies — high risk, high effort.
- The framework lacks guidance on *how* to structure internal packages so they
  map 1:1 to future service boundaries.

## 3. Goals

- G1 — Define a canonical **module structure** inside `internal/` that
  mirrors future service boundaries.
- G2 — Require **explicit module contracts** (interfaces) in `internal/<module>/domain`
  and implementations in `internal/<module>/impl` or `internal/<module>/...`.
- G3 — Enforce **dependency direction**: modules depend on contracts, not on
  other modules' internals; the container wires implementations at the
  composition root only.
- G4 — Provide a **module registry** in the app container to declare module
  providers, enabling `go:generate` or tooling to verify boundaries.
- G5 — Keep the monolith simple: single binary, single container, shared DB
  (or per-module schema), in-process calls — no RPC, no message bus unless
  explicitly added.
- G6 — Document the **extraction checklist** so teams know exactly what
  changes when a module becomes a service.

## 4. Non-Goals

- NG1 — Not a microservice framework: no built-in service mesh, service
  discovery, or distributed tracing.
- NG2 — Not enforcing module boundaries at compile time (Go lacks package
  visibility across modules); this is a *convention* with tooling support.
- NG3 — Not prescribing domain-driven design tactics (aggregates, entities,
  value objects); teams choose their internal patterns.
- NG4 — Not requiring separate databases per module; shared DB with schema
  prefixes is the default, separate DBs are an extraction step.

## 5. Requirements

### 5.1 Canonical Module Layout

Inside `internal/` (per KWF-CCI0N FRK-STR-003), application code organizes
into **modules** — each a candidate for future service extraction.

```
internal/
├── app/                 # composition root (KWF-C9WLJ FRK-CNT-012)
│   └── app.go           # New(cfg) *web.App — wires module providers
├── config/              # typed config (KWF-CCI0N FRK-STR-005)
├── module/
│   ├── catalog/         # example: product catalog module
│   │   ├── domain/      # CONTRACTS — interfaces, DTOs, events (public API)
│   │   │   ├── repository.go   # Repository interface
│   │   │   ├── service.go      # Service interface
│   │   │   ├── events.go       # Domain events
│   │   │   └── dto.go          # Request/response DTOs
│   │   ├── impl/        # IMPLEMENTATION — private, not imported outside
│   │   │   ├── repository.go   # SQL implementation
│   │   │   ├── service.go      # Business logic
│   │   │   └── provider.go     # Module provider (KWF-C9WLJ)
│   │   └── http/        # HTTP handlers depending ONLY on domain interfaces
│   ├── order/           # another module (same structure)
│   └── user/            # another module
└── shared/              # truly shared utilities (no domain logic)
    ├── db/              # DB connection, migrations
    ├── middleware/      # auth, logging, etc.
    └── validation/      # shared validators
```

| ID          | Requirement                                                            | Priority |
| ----------- | ---------------------------------------------------------------------- | -------- |
| FRK-MOD-001 | Application code lives in `internal/<module>/` directories, one per     | Must     |
|             | bounded context. Module names are singular, lowercase (catalog, order). |          |
| FRK-MOD-002 | Each module exposes a **domain** package containing only interfaces,    | Must     |
|             | DTOs, and events — the module's public contract. No implementation     |          |
|             | types leak out of `domain`.                                            |          |
| FRK-MOD-003 | Implementations live in `impl` (or `internal` subpackage) and are       | Must     |
|             | **never imported** by other modules. Only the composition root          |          |
|             | (`internal/app`) and the module's own `http` package may import `impl`. |          |
| FRK-MOD-004 | HTTP handlers (in `module/http`) depend ONLY on `domain` interfaces,   | Must     |
|             | receiving implementations via the container (KWF-C9WLJ).                |          |
| FRK-MOD-005 | Cross-module communication uses **domain events** (published via an     | Should   |
|             | event bus interface in `shared/events`) or direct service calls through |          |
|             | injected interfaces — never direct `impl` imports.                     |          |
| FRK-MOD-006 | A module may declare a `Provider` (KWF-C9WLJ) in `impl/provider.go`     | Must     |
|             | that registers its implementations and HTTP routes.                    |          |

### 5.2 Composition Root Wiring

The `internal/app/app.go` is the **only** place where `impl` packages are
imported. It assembles the container by registering each module's provider:

```go
// internal/app/app.go
func New(cfg *config.Config) (*web.App, error) {
    c, err := rvapp.NewApp(
        &configProvider{cfg: cfg},
        &shared.Provider{},        // DB, middleware, event bus
        &catalog.Provider{},       // catalog module
        &order.Provider{},         // order module
        &user.Provider{},          // user module
        &webProvider{},
        &httpProvider{},
    ).Build()
    ...
}
```

| ID          | Requirement                                                            | Priority |
| ----------- | ---------------------------------------------------------------------- | -------- |
| FRK-MOD-010 | `internal/app` is the **sole** importer of module `impl` packages.     | Must     |
| FRK-MOD-011 | Module providers are registered in dependency order (shared → domain).  | Must     |
| FRK-MOD-012 | The container (KWF-C9WLJ) is the **only** mechanism for cross-module    | Must     |
|             | dependency injection; no global state, no `init()` side effects.       |          |

### 5.3 Shared Kernel

`internal/shared/` holds genuinely shared infrastructure with **no domain
logic**:

- `db/` — database connection pool, migration runner
- `events/` — in-process event bus interface + implementation (for domain
  events between modules)
- `middleware/` — auth, request ID, logging middleware
- `validation/` — shared validators (email, phone, etc.)

Modules import `shared` for infrastructure; `shared` never imports modules.

| ID          | Requirement                                                            | Priority |
| ----------- | ---------------------------------------------------------------------- | -------- |
| FRK-MOD-020 | `internal/shared` contains only infrastructure, no business logic.     | Must     |
| FRK-MOD-021 | Modules depend on `shared` interfaces; `shared` has zero module deps.  | Must     |
| FRK-MOD-022 | An in-process `EventBus` interface lives in `shared/events` for        | Should   |
|             | decoupled module communication.                                        |          |

### 5.4 Extraction Checklist (Non-Functional)

When a module graduates to a service, the following changes are required:

1. Move `internal/<module>/domain` to a **shared Go module** (e.g.,
   `github.com/org/catalog-contracts`) imported by both monolith and
   service.
2. Replace the `impl` package with a **service client** implementing the
   same domain interfaces (HTTP/gRPC client).
3. Register the client in the monolith's container instead of the local
   `impl` provider.
4. Extract the module's **database schema** to a separate database (or keep
   shared with distinct credentials).
5. Add **observability**: distributed tracing headers, metrics, health
   endpoints.
6. Deploy the service independently; update DNS / service mesh config.
7. Run contract tests against both the monolith's client and the service.

| ID          | Requirement                                                            | Priority |
| ----------- | ---------------------------------------------------------------------- | -------- |
| FRK-MOD-030 | Documentation includes a step-by-step extraction checklist.            | Must     |
| FRK-MOD-031 | The checklist maps 1:1 to the module structure (domain → contracts     | Must     |
|             | module, impl → service, http → API gateway).                           |          |

## 6. Non-Functional Requirements

- NFR1 — **Zero runtime overhead**: modular monolith runs as a single
  process with in-process calls; interfaces compile to direct calls.
- NFR2 — **Tooling support**: `krewire verify modules` (future) scans
  imports to detect `impl` leaks across modules.
- NFR3 — **Backward compatible**: existing apps without modules continue
  working; modular structure is opt-in via `krewire new` scaffold.
- NFR4 — **Quality gates**: `gofmt`, `go vet ./...`, `go test ./...` pass
  in every affected repo.

## 7. Success Criteria

- S1 — `krewire new <name>` scaffolds the modular layout with one example
  module (catalog) demonstrating domain/impl/http/provider structure.
- S2 — The example compiles, runs, and serves SSR + API endpoints.
- S3 — No `impl` package is imported outside `internal/app` and its own
  module's `http` package (enforced by `krewire verify modules` or manual
  review).
- S4 — Extraction checklist documented and validated against a sample
  module extraction.

## 8. Related Specifications

| SpecID      | KWF-5ZHQV                              |
| --------- | ----------------------------------------------- |
| [KWF-C4087](./KWF-APP-C4087-krewire-app-framework.md) | Krewire App Framework (assembly) |
| [KWF-C9WLJ](./KWF-DI-C9WLJ-app-container-service-providers.md) | App Container & Service Providers |
| [KWF-CCI0N](./KWF-STRUCT-CCI0N-app-directory-structure.md) | App Project Directory Structure Standard |
| [KWF-230KF](./KWF-HTTP-230KF-http-api-pipeline.md) | HTTP & API Pipeline |

## 9. References

- Modular Monolith pattern: https://github.com/kamilgrzybek/modular-monolith-with-ddd
- "Modular Monoliths" — Simon Brown (2018)
- Go package visibility and internal packages: https://go.dev/doc/modules/managing-dependencies