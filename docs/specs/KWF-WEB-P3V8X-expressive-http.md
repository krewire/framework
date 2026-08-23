# Specification — Expressive HTTP Layer

| Field  | Value |
| ------ | ----- |
| SpecID | KWF-WEB-P3V8X |
| Title  | Expressive HTTP: Routes, Controllers, Request/Response, Middleware |
| Status | Draft |
| Date   | 2026-08-23 |
| Author | Krewire Contributors |
| Domain | Framework — Web — HTTP |

## 1. Context

The current `web` package exposes a solid core (`Router` with `{param}` patterns,
global `Middleware`, `HTTPError` mapping, JSON helpers) but authoring feels
low-level: every handler repeats binding/validation/error plumbing, groups
cannot scope middleware, there are no named routes or controllers, and responses
are written through loose function calls.

This spec layers an expressive surface **on top of** the existing primitives
without breaking them: `Get(pattern, h)` keeps working; new APIs compose.

## 2. Goals

- G1 — Fluent route declarations with names, per-route middleware, and reverse URL generation.
- G2 — Group-scoped middleware (`Group("/admin", requireAdmin)`) without leaking to siblings; global `Use` unchanged.
- G3 — A `Request` value wrapping `*http.Request` + `Params` with typed accessors and one-call binding (JSON body / query) through `libs/validate`.
- G4 — A fluent `Response` builder (status, headers, JSON/Text/HTML/Blob/Redirect) flushed in one `Write`.
- G5 — Generic handlers `H[Req]`/`HQ[Req]`: bind input, invoke a function returning `(any, error)`, map errors via `Error()`, write JSON or a `*Response`.
- G6 — Controllers as plain structs registering their own routes (`Router.Register(ctrl)`).

## 3. Non-Goals

- NG1 — No new router algorithm (linear match kept); no radix tree.
- NG2 — No OpenAPI/codegen; annotations stay out.
- NG3 — No breaking change to `HandlerFunc`, `Get/Post/…`, `Use`, `Group(prefix)` single-arg calls.

## 4. Requirements

### Routing

| ID | Requirement | Priority |
|----|-------------|----------|
| FRK-WEX-001 | `Router.Route(method, pattern) *RouteBuilder` with chained `.Name(string)`, `.Use(mw ...Middleware)`, `.Handle(HandlerFunc)`. Per-route/group middleware wraps the handler receiving `(w, r, Params)` via an adapter that injects matched params. | Must |
| FRK-WEX-002 | `Group(prefix string, mws ...Middleware)` accepts scoped middleware applied only to routes registered through the returned router (nested groups inherit). `Use` remains global. | Must |
| FRK-WEX-003 | Named routes: `.Name` registers segments on the root; `Router.URL(name, PathParams) (string, error)` rebuilds the path, substituting `{param}` values; unknown name or missing param errors. | Must |
| FRK-WEX-004 | `Router.Register(c Controller)` where `Controller` is `interface{ Register(*Router) }`. | Must |

### Request / Response

| ID | Requirement | Priority |
|----|-------------|----------|
| FRK-WEX-010 | `Request` embeds `*http.Request` plus `Params`; `Param(k)`, `Query(k)`, `Bind(dst)` (JSON+validate, reusing `DecodeAndValidate`), `BindQuery(dst)` mapping `query:"name"` fields (string, ints, uint, float, bool, []string) then validate. | Must |
| FRK-WEX-011 | `Response` fluent builder: `Status`, `Set(k,v)`, `JSON(v)`, `Text(s)`, `HTML(s)`, `Blob(ct, b)`, `Redirect(url, code...)`; `Write(http.ResponseWriter)` emits headers before status once. Helpers: `Respond()`, `Created(v)`, `NoContent()`. | Must |

### Generic handlers

| ID | Requirement | Priority |
|----|-------------|----------|
| FRK-WEX-020 | `H[Q any](fn func(*Request, *Q) (any, error)) HandlerFunc` — binds JSON body (empty body ⇒ zero Q), validation failures map to 400 via `Error`, returned `*Response` written as-is, other values written as JSON 200. | Must |
| FRK-WEX-021 | `HQ[Q any](fn func(*Request, *Q) (any, error)) HandlerFunc` — identical but binds from query parameters. | Must |

## 5. Non-Functional

- NFR1 — stdlib + existing deps only; generics require Go ≥ 1.21 (module targets 1.22).
- NFR2 — All existing `web` tests pass untouched; new behavior fully table-tested.
- NFR3 — Zero cost for code paths not using the new layer.

## 6. Success Criteria

- S1 — A controller registers `/users/{id}` under `/api` group with scoped auth middleware; sibling group unaffected; reverse `URL("users.show", …)` yields `/api/users/7`.
- S2 — `HQ` handler binds `?page=2&tag=a&tag=b` into a struct; invalid int → 400 envelope.
- S3 — `gofmt -l .`, `go vet ./...`, `go test ./...` green in `framework`.

## 7. Related Specs

| SpecID | Title |
| ------ | ----- |
| KWF-M07QS | Krewire Web Framework (core this extends) |
