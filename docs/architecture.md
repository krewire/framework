# Architecture — Krewire Framework

## Module Structure (Optimal & Efficient)

Aligned to the unified vision `KWF-M8K2Q` (specs `KWF-T4X9P`/`B7N3D`/`L5H2F`): flat packages, opt-in via import, zero cost when unused.

```
framework/
├── tui/                  # CLI app model — flag/slog/term, `tui.App` harness
├── web/                  # HTTP layer: expressive routes/groups/controllers, request/response, generic handlers; security headers, CSRF/XSS, cache, sessions/cookies; Basic/JWT auth + policy gates; middleware, html/template
│   └── ssg/              # File-based SSG: .kiw DSL (pages/components/layouts) → site/
├── dsl/                  # Kiw DSL (.kiw) — YAML frontmatter + html/template + style/script, Go & JS/TS native
├── test/                 # Test helpers — generic, no spec required
├── ui/                   # Theme, palette, scoped CSS (data-kiw-component/layout)
├── assets/               # Static assets & resources — multi-source Store (dir/embed.FS), ETag/Cache-Control, fingerprint manifest
├── storage/              # App KV storage — Memory/File backends, context-aware, Provider for DI
├── app/                  # Fullstack assembly, DI container, modular monolith (KWF-5ZHQV)
├── runtime/              # WASM client runtime — planned (KWF-T4X9P)
│   ├── js/               # DOM bridge (syscall/js)
│   ├── vdom/             # VNode, diff/patch, RenderHTML/PatchDOM
│   ├── component/        # Component, hooks UseState/UseEffect, registry, hydration
│   ├── widgets/          # Container/Row/Column/Stack, Text/Image, Button/Input, Scaffold/AppBar, ListView
│   ├── layout/           # Flexbox engine (Go)
│   └── style/            # Theme → CSS vars
├── worker/               # Background jobs — planned (KWF-L5H2F)
├── service/              # Microservice patterns — planned (KWF-L5H2F)
│   ├── registry/         # Consul/etcd/NATS/DNS
│   ├── config/           # Distributed config, Watch
│   ├── gateway/          # Route table, middleware, rate limit
│   ├── resilience/       # Circuit breaker, retry, timeout, bulkhead
│   ├── tracing/          # OTel, W3C traceparent
│   └── messaging/        # Publisher/Subscriber/Stream (NATS JetStream)
├── infra/                # Cloud IaC — planned (KWF-B7N3D)
│   ├── provider/         # Provider interface, Plan (pure), Resource
│   ├── schema/           # Canonical kinds: Compute/Database/Storage/Network/DNS/Certificate/SecretRef
│   ├── state/            # Local file + S3/GCS + DynamoDB/Consul locking
│   └── providers/        # aws/ (ECS/Lambda/RDS/S3/CloudFront/Route53) + k8s/ (Deployment/Service/Ingress)
├── framework/              # Meta-package — Name, Version, Banner()
├── docs/                 # Public docs (this folder)
├── examples/
│   ├── greet/            # `cli` minimal example
│   └── app/              # Fullstack example (web+ssg+api)
└── go.mod                # deps: github.com/krewire/libs
```

**Design decisions:**

- **Modular at every Scope (SRP/SoC/High Cohesion).** Industry standard: **Single Responsibility Principle** (SOLID), **Separation of Concerns** (Parnas/Dijkstra), **High Cohesion/Low Coupling** (Constantine), **Unix "Do one thing well"**. Never stack many unrelated funcs in one module; one `Package`/`Func` = one reason to change. Maps to `KWL-ARCH-J2K9Q` `Module→Package→Service→Func` and `KWF-5ZHQV` modular monolith.
- **Import path = workload slice.** `import "github.com/krewire/framework/runtime"` only when frontend interactivity is needed; `app` alone imports `tui`/`web`/`ui`/`app`.
- **Stdlib-first.** `net/http`, `html/template`, `flag`, `log/slog` before third-party; OTel and NATS are the only planned external deps.
- **SSR/hydration parity.** `web/ssg` emits `data-kiw-island` markers; `runtime` hydrates without re-rendering text nodes.
- **Go workspace at hub root.** `go.work` lists all 5 repos (`./framework`, `./libs`, etc.); cross-repo vet/test via `go vet/test ./libs/...` from hub.

## Dependency Graph

```
framework → libs (core, kern, term, config, validate)
         ↘ mdbind ← docs (book kind, file-based routing)
```


## Conventions

- Documentation in English, Markdown, spec-driven (`docs/specs/`); requirements and tests carry `Scope: Workspace/Module/Domain/Package/Service/Func` (`KWL-ARCH-J2K9Q` → `KWL-TEST-P8M4L`).
- Quality gates: `gofmt -l .`, `go vet ./...`, `go test ./...` in each Go repo; per-kind `kiw build` / `kiw build --plan` spot-checks.
- Cross-repo testing via `go.work` workspace (`./framework`, `./libs`, etc.) at hub root; `go work sync` updates `go.work.sum`.
