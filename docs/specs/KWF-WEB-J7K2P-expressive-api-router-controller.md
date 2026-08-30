# Specification — Expressive API Router & Controller Completion

| Field  | Value |
| ------ | ----- |
| SpecID | KWF-WEB-J7K2P |
| Title  | Expressive API Router & Controller Completion — Inspired Ergonomics |
| Status | Draft |
| Date   | 2026-08-30 |
| Author | Krewire Contributors |
| Domain | Framework — Web — HTTP |

## 1. Context

`framework/web` already provides a manual-registry router (`Router` with `{param}` patterns, `Group(prefix,mws)`, `RouteBuilder` with `.Name()` + `.Use()` + `URL()`, `Controller{Register}`, `Module`, `Resource` (KWF-WEB-P3V8X, KWF-WEB-Q8T2R)), generic binders `H[Q]`/`HQ[Q]` + `Request{Bind,BindQuery,Param,Query}` and fluent `Response`. This covers the core for `app`/`service` routing.

Teams building APIs expect a familiar, expressive surface inspired by popular framework ergonomics (fluent verbs, group chaining with prefix/name/middleware, param constraints, resource controllers with conventional names, and a controller base for shared middleware/helpers). The inspiration is ergonomic only — no brand, no magic — idiomatic Go remains the rule.

Krewire's API router is close but incomplete: `Options/Head/Any/Match/Fallback/Redirect` are missing, group name-prefixing (e.g. `name('api.')`-style prefix for nested names) is absent (so a `Resource` on a `/api` group cannot produce `api.users.index` without manual `.Name()` per route), `Resource` is API-only (5 handlers, no `create`/`edit`, names `create` vs `store`, `destroy` alias confusion), and `Controller` is an interface with no base helpers for shared middleware/validation shortcuts.

This spec completes the ergonomic parity strictly for **API routing + controller helpers**, additive on `Router`/`RouteBuilder`/`Resource`/`Controller`, idiomatic Go (no reflection auto-wiring).

## 2. Problem Statement

- **Current pain:** Teams expect `Any`, `Match`, `Options`, `Fallback`, `Redirect` and fluent grouping like `prefix('api')->name('api.')->middleware(auth)->group(fn)`. Without them they hand-roll `Handle` loops, set `NotFound` manually, or duplicate prefix strings per route. `Resource` only exposes 5 handlers (`Index,Show,Create,Update,Delete`) — a full resource (7 routes incl. `create`/`edit` forms) and the API subset are not distinguishable, and group name prefix is missing, so `Group("/v1").Resource("/users")` producing `api.users.*` required a workaround that derived the full path — not composable for arbitrary routes. `Controller` is just `Register(*Router)` with no `BaseController` to own route-scoped middleware or to offer validate/authorize shortcuts next to `H[Q]`.
- **Affected consumers:** `app`/`service` API authors, modular monolith modules (`KWF-5ZHQV`), reviewers auditing route→handler traceability, and teams migrating from fluent route files.
- **Cost of leaving unsolved:** Boilerplate reappears (manual loops over methods), route names drift (no canonical `index/store/show/update/destroy` vs `create` confusion), groups cannot express a name prefix without per-route duplication, and controllers lack a shared home for middleware/auth helpers — each module reinvents its own base.

## 3. Goals

- G1 — Complete the API verb set and shortcuts: `Options`, `Head`, `Any`, `Match`, `Fallback`, `Redirect` (and `PermanentRedirect`).
- G2 — Fluent group chaining: `Prefix(path)`, `Name(prefix)`/`As(prefix)` (name prefix), `Middleware(mws...)`, `Where(param, constraint)` on `RouteBuilder`/`Group`, all composable and preserved through `Routes()`/`URL()`.
- G3 — Complete `Resource` (7 routes) vs `ApiResource` (5 routes) with canonical names, `Only`/`Except` filtering, and backward-compatible `ResourceController` aliases.
- G4 — Provide a Go-idiomatic `BaseController` base with route-scoped middleware registration and response/validate shortcuts that compose with `H[Q]` and `Response`.
- G5 — Additive, zero-cost when unused, stdlib-only, `gofmt`/`go vet`/`go test` green, no breaking change to `Get/Post/Put/Delete/Patch/Handle/Group/Route/Controller`.

## 4. Non-Goals

- NG1 — Reflection-based controller method auto-discovery or implicit model binding.
- NG2 — View routing — covered by `App.Page` + `ssg` for SSR pages.
- NG3 — Trie/radix router — linear match retained (`KWF-WEB-P3V8X NG1`).
- NG4 — `where` regex enforcement at `ServeHTTP` time (constraints stored for introspection only in this phase; matching stays segment equality — enforced later).

## 5. Requirements

### 5.1 Verb Completeness

| ID | Requirement | Priority | Scope |
|----|-------------|----------|-------|
| FRK-LRV-001 | `Router.Options(pattern string, h HandlerFunc)` and `Router.Head(pattern string, h HandlerFunc)` register `OPTIONS`/`HEAD` via `Handle`. | Must | Unit |
| FRK-LRV-002 | `Router.Any(pattern string, h HandlerFunc)` registers the same handler for `GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD`. | Must | Unit |
| FRK-LRV-003 | `Router.Match(methods []string, pattern string, h HandlerFunc)` registers `h` for each method in `methods` (normalized with `strings.ToUpper`; empty `methods` is no-op). | Must | Unit |
| FRK-LRV-004 | `Router.Fallback(h HandlerFunc)` sets `Router.NotFound = h` (catch-all alias). Overwrites any prior `NotFound`. | Must | Unit |
| FRK-LRV-005 | `Router.Redirect(from, to string, code ...int)` registers `GET from` that responds with `302 Found` by default, or the first `code` if supplied. Uses redirect response internally. | Must | Unit |
| FRK-LRV-006 | `Router.PermanentRedirect(from, to string)` registers `GET from` → `301` to `to`. | Should | Unit |

### 5.2 Group Fluency (`prefix` / `name` / `middleware` chaining)

| ID | Requirement | Priority | Scope |
|----|-------------|----------|-------|
| FRK-LRV-010 | `Router.Prefix(prefix string) *Router` is an alias to `Group(prefix)` (inspired by fluent `prefix('admin')`). | Must | Unit |
| FRK-LRV-011 | Router carries a `namePrefix string` (inherited through `Group`/`Prefix`). `Router.Name(prefix string) *Router` and `Router.As(prefix string) *Router` (alias) return a derived router sharing the same `base`/`chain`/`parent` but with `namePrefix` appended (e.g. `r.Name("api.")` or `r.Group("/api").Name("api.")`). Nesting concatenates: `r.Name("a.").Name("b.")` → `"a.b."`. | Must | Unit |
| FRK-LRV-012 | `RouteBuilder.Name(name string)` stores the fully-qualified name `router.namePrefix + name` in both `named` map and `Route.name` field, so `r.Group("/api").Name("api.").Get("/users")` + `.Name("users.index")` yields `api.users.index` and `URL("api.users.index")` resolves to `/api/users`. `DebugString()` and `Routes()` expose the fully-qualified name. | Must | Unit |
| FRK-LRV-013 | `Router.Middleware(mws ...Middleware) *Router` returns `Group("", mws...)` (fluent middleware grouping), enabling `r.Middleware(auth).Group("/admin", ...)` or `r.Prefix("/admin").Middleware(auth).Name("admin.").Group(fn)`. Chaining preserves `base`, `namePrefix`, and `chain`. | Must | Unit |
| FRK-LRV-014 | `RouteBuilder.Where(param, constraint string)` and `Router.Where(param, constraint string) *Router` store `param → constraint` for introspection. Constraints are appended to `RouteInfo.Constraints` and visible via `Routes()`/`DebugString()` as hints but not enforced in `match` (NG4). Group-level `Where` applies to every route registered through the derived router. | Should | Unit |

### 5.3 Resource Completeness (`resource` vs `apiResource` inspired)

| ID | Requirement | Priority | Scope |
|----|-------------|----------|-------|
| FRK-LRV-020 | Extend `ResourceController` to the full 7-action shape while keeping BC aliases: fields `Index HandlerFunc` (`GET /`), `Create HandlerFunc` (`GET /create`), `Store HandlerFunc` (`POST /`), `Show HandlerFunc` (`GET /{id}`), `Edit HandlerFunc` (`GET /{id}/edit`), `Update HandlerFunc` (`PUT/PATCH /{id}`), `Destroy HandlerFunc` (`DELETE /{id}`), plus legacy `Delete HandlerFunc` alias for `Destroy` and `Create` fallback to `Store` when `Store` is nil. | Must | Unit |
| FRK-LRV-021 | `Router.Resource(path string, c ResourceController) *Router` registers the 7 routes with canonical names derived from the full mounted path (`joinPattern(base, path)`): `GET path` → `base.index`, `GET path/create` → `base.create`, `POST path` → `base.store`, `GET path/{id}` → `base.show`, `GET path/{id}/edit` → `base.edit`, `PUT path/{id}` (+ `PATCH`) → `base.update`, `DELETE path/{id}` → `base.destroy`. Any nil handler is skipped. | Must | Unit |
| FRK-LRV-022 | `Router.ApiResource(path string, c ResourceController) *Router` registers the API subset (5 routes): `index, store, show, update, destroy` (skips `create`/`edit`). `store` prefers `Store` else `Create` (BC); `destroy` prefers `Destroy` else `Delete` (BC). Names identical to `Resource` for the 5. | Must | Unit |
| FRK-LRV-023 | `ResourceOptions{Only, Except []string}` with helpers that filter by action name (`index`, `create`, `store`, `show`, `edit`, `update`, `destroy`). At least `Only` and `Except` must be supported for both `Resource` and `ApiResource`. | Should | Unit |

### 5.4 Controller Base (ergonomic, Go-idiomatic)

| ID | Requirement | Priority | Scope |
|----|-------------|----------|-------|
| FRK-LRV-030 | Provide `type BaseController struct{ Middlewares []Middleware }` with `func (b *BaseController) Use(mw ...Middleware)` and `func (b *BaseController) Middleware() []Middleware`. Modules register it as `ctrl.Middleware()` → `r.Group(prefix, ctrl.Middleware()...).Register(ctrl)`. | Must | Unit |
| FRK-LRV-031 | `BaseController` offers response shortcuts delegating to `web.Response`/`web.JSON`: `OK(v)`, `Created(v)`, `NoContent()`, `NotFound(msg)`, `BadRequest(msg)` that return `(*Response, error)` or `*HTTPError` for use inside `H[Q]` handlers. Also `Validate(r *Request, dst any) error` wrapping `Request.Validate`. Document that `BaseController` is optional — plain `struct{ Register(*Router)}` remains minimal `Controller`. | Should | Unit |
| FRK-LRV-032 | Document the ergonomic map: `apiResource`-style registration → `r.ApiResource("/users", ctrl)`, `middleware('auth')->group` → `r.Middleware(auth).Group(...)`, `prefix('admin')->name('admin.')` → `r.Prefix("/admin").Name("admin.")`, `fallback` → `r.Fallback`, `redirect` → `r.Redirect`, `any`/`match` → `r.Any`/`r.Match`. Include in spec and `web/controller.go` doc comment. | Must | Module |

## 6. Non-Functional Requirements

- NFR1 — `gofmt -l .` empty, `go vet ./...` clean, `go test ./...` green in `framework` (`go.work` cross-repo).
- NFR2 — Additive only — existing `Get/Post/.../Group/Route/URL/Resource(Module)` tests pass unchanged; aliases preserve `Destroy`/`Delete`, `Store`/`Create`.
- NFR3 — No new deps, no `unsafe`, linear `ServeHTTP` preserved, manifest `Routes()` O(n).
- NFR4 — `RouteInfo` remains `Method,Pattern,Name,Static` plus optional `Constraints` for `Where` — backward compatible.

## 7. Success Criteria

- S1 — `r.Any("/ping", h)` responds to every method, `r.Match([]string{"GET","POST"}, "/m", h)` responds to exactly those, `r.Options/Head`, `r.Fallback`, `r.Redirect("/old","/new",301)` all behave and appear in `Routes()`/`DebugString()`.
- S2 — `r.Prefix("/api").Name("api.").Middleware(auth).Group(func(g *Router){ g.Get("/users", h).Name("users.index") })` yields route `GET /api/users -> api.users.index` and `URL("api.users.index") == "/api/users"`; `Where("id", "\\d+")` stores constraint visible in `Routes()`.
- S3 — `r.Resource("/photos", ResourceController{Index:..., Store:..., Show:..., ...})` registers 7 routes (or fewer when handlers nil) with canonical names; `r.ApiResource("/photos", ...)` registers exactly 5 (no `create`/`edit`), `Only`/`Except` filters correctly.
- S4 — `BaseController` scopes middleware: embedding and `ctrl.Use(auth)` then `r.Group("/users", ctrl.Middleware()...).Register(ctrl)` works; docs show ergonomic map.

## 8. Related Specifications

| SpecID | Title |
|--------|-------|
| KWF-M07QS | Krewire Web Framework (core routing) |
| KWF-WEB-P3V8X | Expressive HTTP: Routes, Controllers, Request/Response, Middleware |
| KWF-WEB-Q8T2R | Manual Registry Router — Code-First, Modular Routing |
| KWF-WEB-R9T4C | HTTP Security & State |
| KWF-5ZHQV | Modular Monolith Architecture Default |

## 9. References

- Current impl: `framework/web/router.go` (Handle/Group/RouteBuilder/URL), `route.go` (named map), `registry.go` (Resource/Module), `request.go` (`H[Q]`/`HQ`), `response.go` (`Respond`).
