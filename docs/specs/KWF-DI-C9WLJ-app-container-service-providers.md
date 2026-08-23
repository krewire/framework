# Specification — App Container & Service Providers

| Field       | Value                                       |
| ----------- | ------------------------------------------- |
| SpecID      | KWF-C9WLJ                              |
| Title       | App Container & Service Providers           |
| Status      | Draft                                       |
| Date        | 2026-08-19                                  |
| Author      | Krewire Contributors                         |
| Domain      | Framework — Dependency Container / DI       |

## 1. Context

Web `App` assembly (KWF-C4087) composes routes, pages, and assets, but it
does not answer "who builds the data layer, the config, the clients, and in
what order?". Applications growing beyond a hello-page need a **composition
root** — one place that defines every dependency, how it is constructed, and
how it can be swapped. Modern frameworks provide exactly this with a service
container and service providers (Laravel's container + providers, Spring's
application-context + beans, SvelteKit's context).

This spec defines the Krewire **app container** and **service providers**:
the idiomatic Go mechanism to centralize application control; enable
dependency injection; make construction order and failures observable;
allow data mocking; and keep the assembly deterministic and testable. It
lives in the framework (`app` package) and integrates with the canonical
layout (KWF-CCI0N), the `web.App` (KWF-C4087), and configuration/validation
(KWL-2X1QZ, KWL-LHANF).

## 2. Problem Statement

Without a container, application wiring spreads across `main` and callers:
services are constructed inside handlers or as globals, order is implicit,
construction is untestable, and swapping a real store for a fake requires
editing every usage site. Observability is ad-hoc because no single place
knows "what was built, in which order, and how long it took". Providers
(handlers, APIs, background tasks) each reinvent their own startup logic.

The app container solves this by making every dependency a **typed binding**
that is built and resolved at a single, explicit composition root.

## 3. Goals

- G1 — Provide a typed, explicit dependency container (no global state, no
  reflection-driven discovery) as the single composition root of an app.
- G2 — Provide a service-provider abstraction that groups registration and
  bootstrapping, with a deterministic, documented order.
- G3 — Make construction observable: resolution order, durations, and
  failures are structured records (predictable observability).
- G4 — Make swapping real dependencies for fakes/mocks a one-line, typed
  operation (data mocking, test seams, env-selected variants).
- G5 — Fail fast: any unresolvable or circular dependency aborts startup
  with a full resolution-path error before the server listens.
- G6 — Keep dependency direction: framework → `libs`; `app` container never
  imports `web`, it *receives* the constructed `web.App` as a binding.

## 4. Non-Goals

- NG1 — Reflection-based auto-wiring from struct tags or filesystem scanning
  (protected by KWF-C4087 FRK-APP-004).
- NG2 — A runtime "hive" or plugin/kernel system; providers are plain Go.
- NG3 — Orchestrating multiple processes; this serves the single-process
  monolith.
- NG4 — Replacing good constructor injection; the container is the *default*
  composition root, packages keep plain exported constructors (e.g.
  `store.New(cfg)`).

## 5. Requirements

### 5.1 App Container Core

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-CNT-001 | Provide a container as the sole composition root: `app.New()` returns a container; bindings are registered before build and never after first resolution. | Must |
| FRK-CNT-002 | Bindings are typed and keyed by Go type only — `app.Provide[T](factory)`, `app.Singleton[T](factory)`; no string names, no `any` lookups in normal use. | Must |
| FRK-CNT-003 | `Provide` builds a fresh value per resolve; `Singleton` builds lazily once and reuses it, safe under concurrent access. | Must |
| FRK-CNT-004 | `app.Resolve[T]() (T, error)` returns the typed dependency; a missing binding, unresolvable factory, or constructor error yields an error carrying the full resolution path. | Must |
| FRK-CNT-005 | Resolution is deterministic: the build order is fixed by registration order; identical registrations always resolve identically. | Must |
| FRK-CNT-006 | Circular dependencies are detected and reported with the cycle path; they abort build, not hang. | Must |
| FRK-CNT-007 | A factory is an ordinary function — `func(deps...) (T, error)` — whose parameters are dependencies resolved from the container; no magic tags. | Must |
| FRK-CNT-008 | The container offers an `Override[T]` replacing an existing binding before build, used for mocks, tests, and environment variants. | Must |
| FRK-CNT-009 | After the first resolution the container locks; later `Provide`/`Override` calls fail loudly (misuse guard). | Should |
| FRK-CNT-010 | Container and bindings require no external non-stdlib dependency beyond the Krewire `libs` the app already uses. | Must |

### 5.2 Service Providers

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-CNT-011 | Define a `Provider` interface grouping an app's control: a `Register(c *Container)` phase that declares bindings and a `Boot(c *Container) error` phase that runs after all bindings exist. | Must |
| FRK-CNT-012 | `app.App` (or equivalent) aggregates providers: `New(providers...)`, runs every `Register` in order, then every `Boot` in order, then returns the container for resolution. | Must |
| FRK-CNT-013 | Provider order is explicit and stable (call order); it is the documented contract for construction and startup sequencing. | Must |
| FRK-CNT-014 | `Boot` may resolve dependencies to validate wiring and run initializers; any returned error aborts startup with a structured failure naming the provider. | Must |
| FRK-CNT-015 | A provider that requires runtime configuration declares it through `libs/config` + `libs/validate` bindings (FRK-STR-005), so config/validation errors surface during `Register`/`Boot`, before serving. | Must |
| FRK-CNT-016 | Providers live in the canonical layout: assembly in `internal/app`, additional providers under `internal/app/<group>` or `internal/providers/` (KWF-CCI0N). | Should |

### 5.3 Dependency Injection & Composition

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-CNT-020 | A web provider binds the built `*web.App` (KWF-C4087) into the container; `main` only resolves it and calls `Run` — no wiring logic in `main`. | Must |
| FRK-CNT-021 | Handlers and services accept their dependencies via constructor/factory parameters resolved by the container; they never reach for globals or `sync.Once` singletons. | Must |
| FRK-CNT-022 | The container supports composing groups (e.g. all `Middleware`, all jobs) as named-order slices resolved together, without string lookups leaking into call sites. | Should |
| FRK-CNT-023 | Configuration, storage, HTTP, pages, and embedding each ship as a provider so a provider's responsibilities have a single home. | Should |

### 5.4 Predictable Observability

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-CNT-030 | Every successful resolve emits a structured record via `log/slog` containing the bound type, duration, and whether it was a singleton cache hit. | Must |
| FRK-CNT-031 | A boot summary is emitted at the end of startup: provider order, resolved types, total startup duration — stable and diffable across runs (predictable observability). | Must |
| FRK-CNT-032 | Failed resolutions emit the resolution path and the provider that owns the failing binding, in one structured error. | Must |
| FRK-CNT-033 | Observability is opt-in per category (resolution trace, boot summary) via container flags, defaulting to a concise boot summary on. | Should |
| FRK-CNT-034 | Hooks (`BeforeResolve`/`AfterResolve`, or a `Tracer` interface) allow apps to attach metrics/metering without changing provider code. | Should |

### 5.5 Mocking & Testability

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-CNT-040 | `Override[T]` is the canonical mock seam: tests build the app container wired to fakes (in-memory store, stubbed HTTP client) with zero provider edits. | Must |
| FRK-CNT-041 | Provide a test helper that assembles an app (`app.New(...).Test(overrides...)`) with overrides applied before build and a short-circuit boot. | Must |
| FRK-CNT-042 | Environment-selected variants (e.g. in-memory vs. real store) are expressed as provider-level conditionals selecting which factories to register, using `libs/config` env overrides. | Should |
| FRK-CNT-043 | Data mocking is typed: a fake binding satisfies the same interface Type `T` as the real one, so consumers are tested unchanged. | Must |

## 6. Non-Functional Requirements

- NFR1 — **Memory safety.** The `unsafe` package must not be used.
- NFR2 — **Dependencies.** `app` uses stdlib + `libs` (`core`, `config`, `validate`, `log`); it never imports `web` or project packages.
- NFR3 — **Portability.** Linux, macOS, Windows.
- NFR4 — **Quality gates.** `gofmt`, `go vet ./...`, `go test ./...` pass in affected repos.
- NFR5 — **Determinism.** Identical registrations produce identical resolution order, boot summary, and binary output.
- NFR6 — **Fail-fast.** All wiring errors surface before `*web.App` starts listening.

## 7. Success Criteria

- S1 — `examples/app` (or a scaffolded app) splits wiring into providers and resolves `*web.App` from the container; behavior is unchanged.
- S2 — A test swaps the store with an in-memory fake via `Override[T]` without editing any provider.
- S3 — Startup prints a date-stable boot summary: provider order and resolved types; a missing binding prints its resolution path.
- S4 — Startup fails (non-zero exit) before listening when a binding is missing or circular.
- S5 — Quality gates pass across `framework` and `libs`.

## 8. Related Specifications

| SpecID      | KWF-C9WLJ                              |
| --------- | ----------------------------------------------- |
| [KWF-CCI0N](./KWF-STRUCT-CCI0N-app-directory-structure.md) | App Project Directory Structure Standard |
| [KWF-C4087](./KWF-APP-C4087-krewire-app-framework.md) | Krewire App Framework (assembly) |
| [KWF-PPUWX](./KWF-UI-PPUWX-layout-ui-system.md) | Layout & UI System |
| [KWF-230KF](./KWF-HTTP-230KF-http-api-pipeline.md) | HTTP & API Pipeline |
| [KWF-0F2EB](./KWF-WEB-0F2EB-server-frontend-pipeline.md) | Server & Frontend Rendering Pipeline |
| [KWF-F2TQC](./KWF-JS-F2TQC-js-ts-framework-integration.md) | JS/TS Framework Integration |
| [KWN-6K41E](https://github.com/krewire/krewire/blob/main/docs/specs/KWN-RUN-6K41E-krewire-run-dev-deploy.md) | krewire run/dev/deploy |
| [KWL-2X1QZ](https://github.com/krewire/libs/blob/main/docs/specs/KWL-CONFIG-2X1QZ-configuration-loading.md) | Configuration Loading |
| [KWL-LHANF](https://github.com/krewire/libs/blob/main/docs/specs/KWL-VALIDATE-LHANF-struct-validation.md) | Struct Validation |

## 9. References

- [KWF-CMBZJ](./KWF-META-CMBZJ-krewire-meta-framework.md) — Krewire Meta-Framework (module architecture; libs are framework-agnostic).
- Go standard library `os/signal`, `log/slog`, and constructor-injection idioms.