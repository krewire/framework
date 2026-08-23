# Specification — HTTP & API Pipeline

| Field       | Value                                       |
| ----------- | ------------------------------------------- |
| SpecID      | KWF-230KF                              |
| Title       | HTTP & API Pipeline                         |
| Status      | Draft                                       |
| Date        | 2026-08-19                                  |
| Author      | Krewire Contributors                         |
| Domain      | Frameworks — Web / HTTP                     |

## 1. Context

The `web` package (KWF-M07QS) provides routing, template rendering, and static
export on `net/http`. To power fullstack monolith apps, it needs an **HTTP &
API layer**: a middleware chain for cross-cutting concerns (auth, logging,
recovery), a JSON contract for API endpoints, structured error mapping, and
request decoding that reuses `libs/validate` for typed DTOs.

This pipeline is the "backend/API" half of the monolith. It is additive to the
existing `web` package and never forces a project to use it (a pure static site
keeps working untouched).

## 2. Problem Statement

A fullstack app needs more than routes: it must enforce middleware before every
handler, speak JSON with callers, decode and validate request bodies, and turn
domain errors into correct HTTP responses. Without this layer, each app
re-implements:

- Middleware wiring around `net/http` (logging, panic recovery, auth gates).
- JSON encoding/decoding conventions (content-type, error shape).
- Error→status mapping (validation becomes 400, not-found 404, etc.).

The result is inconsistent APIs, unsafe decoders, and duplicate plumbing across
Krewire apps. A shared pipeline fixes this once in the framework.

## 3. Goals

- G1 — Provide a `net/http`-compatible middleware chain composable with stdlib handlers.
- G2 — Provide JSON encode/decode helpers with strict, bounded decoding.
- G3 — Integrate `libs/validate` so request DTOs validate automatically in one call.
- G4 — Provide an error type carrying an HTTP status with deterministic error→status mapping.
- G5 — Keep the API purely additive to `web`: routing, templates, and export behavior unchanged.
- G6 — Build on the standard library + `libs` only.

## 4. Non-Goals

- NG1 — A full REST/CRUD generator or automatic handler scaffolding.
- NG2 — A dependency-injection container or service locator.
- NG3 — Sessions, authentication primitives, or authorization frameworks.
- NG4 — OpenAPI / JSON-Schema generation from Go types in this phase.
- NG5 — WebSockets or streaming protocols in this pipeline.
- NG6 — A custom JSON codec; `encoding/json` is the wire format.

## 5. Requirements

### 5.1 Middleware

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-API-001 | Provide `type Middleware func(next http.Handler) http.Handler` compatible with stdlib middleware. | Must |
| FRK-API-002 | `Router.Use(mw ...Middleware)` registers middleware applied in registration order to every route. | Must |
| FRK-API-003 | Provide `MiddlewareChain(mws ...Middleware) Middleware` composing left-to-right with a final handler. | Must |
| FRK-API-004 | Provide built-in `RecoverMiddleware` (logs panics via `slog`, returns 500, no stack leak to client). | Must |
| FRK-API-005 | Provide built-in `AccessLogMiddleware` (structured method, path, status, duration via `slog`). | Should |
| FRK-API-006 | A handler may use the request context to propagate values set by middleware. | Must |

### 5.2 JSON Contract

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-API-010 | Provide `JSON(w, status int, v any)` writing the value as JSON with `Content-Type: application/json`. | Must |
| FRK-API-011 | Provide `ReadJSON(r *http.Request, dst any) error` decoding the body with a bounded reader and one trailing-value check. | Must |
| FRK-API-012 | `ReadJSON` returns a wrapped error classifying malformed JSON (`ErrInvalidJSON`). | Must |
| FRK-API-013 | Provide `DecodeAndValidate(r *http.Request, dst any) error` that decodes then validates `dst` via `libs/validate`. | Must |
| FRK-API-014 | Empty bodies decode into zero `dst`; unknown fields are rejected on the top-level object. | Should |

### 5.3 Errors & Status

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-API-020 | Provide `HTTPError{Status int, Code string, Message string, Details any}`; it implements `error`. | Must |
| FRK-API-021 | Provide `Error(w, err error)` mapping: validation errors → 400, `HTTPError` → its status, else 500. | Must |
| FRK-API-022 | Error responses are JSON: `{"code","message","details"}` with a stable shape. | Must |
| FRK-API-023 | Provide canonical helpers `BadRequest`, `NotFound`, `Forbidden`, `Internal`, `Conflict` returning `*HTTPError`. | Must |
| FRK-API-024 | `errors.Is`/`errors.As` work with `HTTPError` for unwrapping in tests. | Should |

### 5.4 Routing Ergonomics

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-API-030 | Route params (`{name}`) remain available via `web.Params` in handlers. | Must |
| FRK-API-031 | Query strings remain readable via `r.URL.Query()`; no binding layer in this phase. | Must |
| FRK-API-032 | Provide a `Group(prefix string)` helper returning a scoped router sharing the parent's middleware, for versioned APIs (`/api/v1`). | Should |
| FRK-API-040 | The JSON contract must be stable and documented (field naming, error shape), so client-side frameworks and hydration payloads (`data-props`, KWF-F2TQC) can consume the same shape. | Must |

## 6. Non-Functional Requirements

- NFR1 — **Memory safety.** The `unsafe` package must not be used.
- NFR2 — **Dependencies.** `framework/web` depends only on stdlib + `libs` (`core`, `validate`).
- NFR3 — **Performance.** Middleware adds no per-request allocation on the happy path; decoding is bound (e.g. 1 MiB).
- NFR4 — **Quality gates.** `gofmt`, `go vet ./...`, `go test ./...` pass.
- NFR5 — **Determinism.** Identical input yields identical JSON and error output.

## 7. Success Criteria

- S1 — A test app mounts middleware, a `/api/*` JSON endpoint, and a validating DTO; bad input returns 400 with a stable body.
- S2 — A panicking handler returns 500 through `RecoverMiddleware` and the process does not crash.
- S3 — `DecodeAndValidate` rejects a malformed body and a validation failure as distinct, wrapped errors.
- S4 — Exported static pages from KWF-PT8OD build unchanged when this pipeline is linked.

## 8. Related Specifications

| SpecID      | KWF-230KF                              |
| --------- | ----------------------------------------------- |
| [KWF-M07QS](./KWF-WEB-M07QS-krewire-web-framework.md) | Krewire Web Framework (host) |
| [KWF-0F2EB](./KWF-WEB-0F2EB-server-frontend-pipeline.md) | Server & Frontend Rendering Pipeline |
| [KWF-C4087](./KWF-APP-C4087-krewire-app-framework.md) | Krewire App Framework (monolith assembly) |
| [KWF-F2TQC](./KWF-JS-F2TQC-js-ts-framework-integration.md) | JS/TS Framework Integration (shared JSON/data-props contract) |
| [KWL-LHANF](https://github.com/krewire/libs/blob/main/docs/specs/KWL-VALIDATE-LHANF-struct-validation.md) | Struct Validation (dependency) |
| [KWL-W0J2X](https://github.com/krewire/libs/blob/main/docs/specs/KWL-CORE-W0J2X-errors-exit-codes.md) | Core Errors & Exit Codes |

## 9. References

- [KWF-PPUWX](./KWF-UI-PPUWX-layout-ui-system.md) — Layout & UI System (component registry the SSR layer uses).
- [KWF-PT8OD](./KWF-SSG-PT8OD-static-site-generator.md) — Static Site Generator.