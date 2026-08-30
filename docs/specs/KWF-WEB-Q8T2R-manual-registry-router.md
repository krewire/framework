# Specification — Manual Registry Router

| Field  | Value |
| ------ | ----- |
| SpecID | KWF-WEB-Q8T2R |
| Title  | Manual Registry Router — Code-First, Modular Routing for Large Projects |
| Status | Draft |
| Date   | 2026-08-30 |
| Author | Krewire Contributors |
| Domain | Framework — Web — HTTP |

## 1. Context

Krewire ships two routing surfaces today:

- **App/Router** (`framework/web`): code-first — `r.Get("/users/{id}", h)`, `r.Group("/api", mw)`, `RouteBuilder` with `.Name()` + `URL()`, and `Controller { Register(*Router) }`. This is a manual registry already.
- **SSG file-based pipeline** (`KWF-DF3PL`, `web/ssg/LoadFromDir`): `pages/*.kiw`, `components/*.kiw`, `layouts/*.kiw`, `content/*/`, `public/` derive routes from the filesystem. Ideal for marketing sites and books where URL ≈ file tree.

For large `app`/`service` projects (KWF-5ZHQV modular monolith) the file-based SSG convention — when applied beyond its sweet spot — creates the exact pain the request describes: one route = one file, deep directory trees mirroring URL trees, layout wiring duplicated per file, no code-searchable route table, and no module boundary (a `catalog` domain cannot co-locate its routes, handlers, and middleware without scattering files across `pages/catalog/*`).

KWF-DF3PL is intentionally Astro-like for `site`/`book` kinds. Large `app`/`service` workloads need the opposite: a single, code-searchable registry that is **module-scoped** (`internal/catalog/http/routes.go` registers `catalog` routes) and composes at the `internal/app` composition root.

This spec formalizes the **Manual Registry Router** — a code-first complement to file-based routing, not a replacement. `site`/`book` stay file-based; `app`/`service`/`worker`-adjacent HTTP surfaces use the manual registry. Both can coexist in one repo (hybrid).

## 2. Problem Statement

- **Current pain:** Teams that start with `pages/` for a large app discover late that file-based routing does not modularize. Adding one REST resource fans out to 5+ files/dirs (`pages/api/users/index.kiw`, `pages/api/users/[id].kiw`, …), each with duplicated `layout:` frontmatter and scoped style boilerplate. Renaming a route requires renaming files and fixing imports. Middleware cannot be scoped per domain without wrapping whole directory trees. The route table is implicit (filesystem) — `grep` cannot answer "which handler serves GET /api/users/{id}?".
- **Affected consumers:** `app`/`service` authors building modular monoliths (`KWF-5ZHQV`), reviewers auditing route→handler traceability, and tooling that wants to list or lint routes.
- **Cost of leaving unsolved:** Boilerplate scales linearly with routes, directory structure becomes the architecture, domain code leaks across `pages/` instead of living in `internal/<module>/http`, and extraction to a service (FRK-MOD-030) requires first untangling a file tree. Developers revert to ad-hoc `http.ServeMux` or hand-rolled registrars outside the framework.

The project **does have this problem today**: `web.Router` exists but lacks the affordances large projects rely on — route introspection, duplicate/conflict detection, a `Module` mounting primitive, and a `Resource` helper — so teams either underuse it or wrap it inconsistently.

## 3. Goals

- G1 — Make manual registry the **canonical** routing style for `app`/`service`; file-based stays canonical for `site`/`book`.
- G2 — Provide **module-scoped registration** — one `internal/<module>/http` package registers its routes/middleware via a `Registrar`/`Module` that mounts under a prefix at the composition root with zero filesystem coupling.
- G3 — Expose **route introspection** — `Router.Routes()`, `HasRoute`, `RouteInfo` — so the route table is searchable, printable (`DebugString`), and testable.
- G4 — Detect **duplicates/conflicts** early — registering the same `METHOD pattern` twice is an error surfaced at registration time (panic in `MustHandle`, error return in `HandleChecked`/`Route().Handle` remains last-wins for BC but is lintable via `Routes()`).
- G5 — Reduce REST boilerplate via **Resource helper** — `r.Resource("/users", ctrl)` expands to the 5 conventional REST routes with named routes.
- G6 — Keep **zero-cost and backward compatible** — existing `Get/Post/Handle/Group/Controller/RouteBuilder/URL` keep working; new APIs are additive, stdlib-only, `go fmt/vet/test` green.

## 4. Non-Goals

- NG1 — Replacing file-based SSG (`KWF-DF3PL`) or unifying it into one engine; file-based stays as-is for `site`/`book`.
- NG2 — A radix tree / trie router; linear match is kept (KWF-WEB-P3V8X NG1), manifest iterates the same slice.
- NG3 — OpenAPI/codegen, annotation-driven routing, or reflection-based auto-registration.
- NG4 — Compile-time enforcement of `internal/<module>/impl` boundaries; that remains `KWF-5ZHQV` convention + future `krewire verify modules`.

## 5. Requirements

### 5.1 Registry Primitives (additive, no breaking change)

| ID | Requirement | Priority | Scope |
|----|-------------|----------|-------|
| FRK-REG-001 | `Router` retains `HandlerFunc func(http.ResponseWriter,*http.Request,Params)`, `Get/Post/Put/Delete/Patch/Handle`, `Group(prefix, mws...)`, `Use(mw...)`, `Route(method,pattern) *RouteBuilder`, `Register(Controller)`, `URL(name, params)` as-is (BC). | Must | Unit |
| FRK-REG-002 | **Registrar alias:** `type Registrar interface { RegisterRoutes(r *Router) }` (or `Controller` alias) — either name is accepted; `Router.Register(Registrar)` and `Router.Mount(Registrar)` are aliases that call `RegisterRoutes`/`Register`. Document that `Controller` is the historical name (`KWF-WEB-P3V8X FRK-WEX-004`). | Must | Unit |
| FRK-REG-003 | **Module mounting:** Provide `type Module struct { Prefix string; Middleware []Middleware; Register func(*Router) }` with `func (m Module) RegisterRoutes(r *Router)` that creates `g := r.Group(m.Prefix, m.Middleware...)` then invokes `m.Register(g)` if non-nil. Modules compose at `internal/app/app.go` as `r.Register(module.Routes)` or `r.Register(Module{Prefix:"/catalog", ...})`. | Must | Unit |
| FRK-REG-004 | **Mount shorthand:** `Router.Mount(prefix string, fn func(*Router), mws ...Middleware)` equivalent to `Group(prefix,mws...).` + `fn`. Convenience for `internal/app` without a struct. | Should | Unit |

### 5.2 Route Introspection & Duplicate Detection

| ID | Requirement | Priority | Scope |
|----|-------------|----------|-------|
| FRK-REG-010 | Store the original pattern string plus optional name on each registered route (internal fields on `Route`: `pattern`, `name`). | Must | Unit |
| FRK-REG-011 | `type RouteInfo struct { Method, Pattern, Name string; Static bool }` plus `func (r *Router) Routes() []RouteInfo` returning a copy ordered by registration (global + groups flattened, `joinPattern` resolved). Static routes (`Static/StaticFS`) may appear as `Method:"STATIC"`. | Must | Unit |
| FRK-REG-012 | `func (r *Router) HasRoute(method, pattern string) bool` — reports whether a method+pattern already registered (pattern normalized via `joinPattern` relative to router base; for a group, checks against its root). | Must | Unit |
| FRK-REG-013 | `func (r *Router) RouteExists(method, pattern string) bool` alias of `HasRoute` for discoverability (one must exist). | Should | Unit |
| FRK-REG-014 | **Duplicate detection:** `Handle`/`Route().Handle` keep last-wins for BC, but `Router` records duplicates; `Routes()` exposes them and a helper `func (r *Router) CheckDuplicates() []RouteInfo` (or `Validate() error`) returns the second+ occurrences. `MustHandle(method,pattern,h)` panics on duplicate for strict registries. | Must | Unit |
| FRK-REG-015 | `func (r *Router) DebugString() string` — human-readable manifest (`GET  /users/{id}  -> users.show`) for `krewire info` / tests, sorted by method then pattern for determinism in output (registration order preserved in `Routes()`). | Should | Unit |

### 5.3 Resource Helper (REST boilerplate reduction)

| ID | Requirement | Priority | Scope |
|----|-------------|----------|-------|
| FRK-REG-020 | `type ResourceController struct { Index, Show, Create, Update, Delete HandlerFunc }` — any nil field is skipped. | Must | Unit |
| FRK-REG-021 | `func (r *Router) Resource(path string, c ResourceController) *Router` registers: `GET path` → `Index` (name `path.index`), `GET path/{id}` → `Show` (`path.show`), `POST path` → `Create` (`path.create`), `PUT path/{id}` → `Update` (`path.update`), `DELETE path/{id}` → `Delete` (`path.destroy`). `path` is a prefix like `"/users"` (no trailing slash). Names derive from `strings.Trim(path,"/")` with `"/"` → `"resource"` base; dots sanitized (`"/api/users"` → `"api.users.index"`). | Must | Unit |
| FRK-REG-022 | `Resource` respects `Router.base` (when called on a `Group`) so `api.Group("/v1").Resource("/users", ...)` mounts under `/api/v1/users`. | Must | Unit |

### 5.4 Page Registry (App-level manual, complements SSG)

| ID | Requirement | Priority | Scope |
|----|-------------|----------|-------|
| FRK-REG-030 | `App` already has `Page(PageSpec)`, `Component`, `Layout`, `Asset`, `Theme`, `Router()` — document that **App is the manual registry for pages**: large projects do not use `LoadFromDir`; they call `app.Page(...)` per route in module registrars and mount APIs via `app.Router().Register(...)`. No new `PageRegistry` type is required; expose `App.Routes()` delegating to `Router.Routes()` plus page count for manifest parity. | Must | Module |
| FRK-REG-031 | Example wiring for modular monolith in docs/specs: `internal/catalog/http/routes.go` exports `func Register(r *web.Router, svc domain.Service)` or `var Module = web.Module{Prefix:"/catalog", Register: func(r *web.Router){...}}`; `internal/app/app.go` does `app.Router().Register(cataloghttp.Module)` etc., keeping `internal/app` as the sole importer of `impl`. | Must | Module |

### 5.5 Interop with File-Based SSG

| ID | Requirement | Priority | Scope |
|----|-------------|----------|-------|
| FRK-REG-040 | Document the decision matrix: `site`/`book` → file-based (`ssg.LoadFromDir` + `Site.Build/Handler`); `app`/`service` → manual registry (`web.App` + `Router`); hybrid (marketing pages + app) → `ssg.Site` for static + `web.Router` for APIs, sharing `ui.Registry` for components. No runtime fallback that auto-scans `pages/` when manual registry is in use. | Must | Module |
| FRK-REG-041 | `go vet`/`go test` remain the quality gates; no `unsafe`, no new deps. | Must | Module |

## 6. Non-Functional Requirements

- NFR1 — `gofmt -l .`, `go vet ./...`, `go test ./...` pass in `framework` (and `go test ./...` at workspace root via `go.work`).
- NFR2 — Additive only — existing tests (`TestFRK_WEX_*`, `TestFRK_WEB_*`) pass unchanged.
- NFR3 — No performance regression — `ServeHTTP` stays linear, manifest is O(n) copy.
- NFR4 — `krewire info` / `krewire build` behavior unchanged for `site`/`book`; manual registry is a library feature, not a CLI flag.

## 7. Success Criteria

- S1 — A modular monolith with 3 modules (`catalog`, `order`, `user`) registers all routes via `Module`/`Registrar` at `internal/app/app.go` without any `pages/` directory; `Routes()` lists every `METHOD pattern [name]`, `DebugString()` prints a deterministic manifest, `HasRoute`/`CheckDuplicates` detect a duplicate `GET /users/{id}`.
- S2 — `r.Resource("/users", ResourceController{Index:..., Show:...})` registers 2 routes, skips the 3 nil ones, and `URL("users.show", PathParams{"id":"7"})` → `"/users/7"` (or `"/api/users/7"` when mounted on a group).
- S3 — A `site` project continues to build via `ssg.LoadFromDir` untouched; an `app` project uses `web.NewApp().Page(...)` + `Router()` manual pages; hybrid docs show how to share `ui.Default()`.
- S4 — `gofmt -l .` empty, `go vet ./...` clean, `go test ./...` green in `framework`.

## 8. Related Specifications

| SpecID | Title |
|--------|-------|
| KWF-M07QS | Krewire Web Framework (core routing this extends) |
| KWF-WEB-P3V8X | Expressive HTTP: Routes, Controllers, Request/Response, Middleware |
| KWF-DF3PL | File-Based Site Pipeline (`.kiw` Modules) — stays canonical for `site`/`book` |
| KWF-5ZHQV | Modular Monolith Architecture Default — module wiring target |
| KWF-CCI0N | App Project Directory Structure Standard |
| KWF-WEB-P3V8X | Expressive HTTP (FRK-WEX-001..004) — `RouteBuilder`, `Group`, `Register` |

## 9. References

- Current router: `framework/web/router.go` (256 lines), `route.go` (119 lines) — `ApplyChain`, `joinPattern`, `parsePattern`, `named` reverse map.
- Modular monolith extraction checklist: `KWF-5ZHQV §5.4`.
- File-based SSG authoring: `KWF-DF3PL §5 (FRK-FLS-*)`.
