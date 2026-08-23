# Krewire Framework

**Krewire Framework** is the unified Go framework for every web-service workload. One import path powers CLI tools, HTTP backends, static sites, documentation sites, reactive frontends (Go→WASM), fullstack monoliths, background workers, microservices, and cloud infrastructure.

It is the engine behind [`mdbind`](https://github.com/krewire/mdbind), [`krewire`](https://github.com/krewire/krewire), and every Krewire project built by `krewire new` / `krewire init`.

> Unified vision: [`KWF-M8K2Q`](docs/specs/KWF-ARCH-M8K2Q-unified-framework-vision.md) — with WASM runtime [`KWF-T4X9P`](docs/specs/KWF-WASM-T4X9P-wasm-client-runtime.md), cloud infra [`KWF-B7N3D`](docs/specs/KWF-INFRA-B7N3D-cloud-provider-abstraction.md), and microservice/worker [`KWF-L5H2F`](docs/specs/KWF-SVC-L5H2F-microservice-patterns.md).

## Workload Coverage

| Workload | Package | Status |
|----------|---------|--------|
| CLI tools | `cli` — App, Command, flag/slog/term integration | ✅ Shipped |
| Backend / HTTP API | `web` — Router, middleware, `web.App` | ✅ Shipped |
| Static sites (SSG) | `web/ssg` — declarative layouts/components/pages + assets | ✅ Shipped |
| Theming & UI | `ui` — Theme, palette, scoped CSS | ✅ Shipped |
| Fullstack / Monolith | `app` — assembly + DI container, modular layout | ✅ Shipped |
| Frontend (WASM) | `runtime` — Go→WASM, VDOM, widgets, hydration islands | 🔜 KWF-T4X9P |
| Workers | `worker` — queues, cron, retries, DLQ | 🔜 KWF-L5H2F |
| Microservice | `service` / `web/gateway` — registry, gateway, resilience, tracing | 🔜 KWF-L5H2F |
| Cloud Infra | `infra` — provider abstraction, state/locking, AWS + Kubernetes | 🔜 KWF-B7N3D |

All workloads share a single `krewire.yaml` configuration (validated by [`libs`](https://github.com/krewire/libs)) and a single CLI (`krewire`).

## Package Layout

```
framework/
├── tui/        # CLI application model
├── web/        # HTTP server, routing, middleware
│   └── ssg/    # File-based SSG (Astro-inspired, .kiw DSL)
├── dsl/        # Kiw DSL (.kiw) — frontmatter YAML + html/template + scoped CSS, parseable to Go & JS/TS
├── ui/         # Theme, palette, scoped CSS
├── app/        # Fullstack assembly + DI container
├── test/       # Test helpers (generic, spec-driven opt-in)
├── runtime/    # Client runtime (WASM) — planned
├── worker/     # Background jobs — planned
├── service/    # Microservice patterns — planned
├── infra/     # Cloud provider abstraction — planned
├── framework/  # Meta-package re-exports (if applicable)
└── examples/   # Runnable examples (greet, app)
```

Related primitives live in [`libs`](https://github.com/krewire/libs) (`core`, `term`, `config`, `validate`).

## Getting Started

### Prerequisites

- Go 1.22+ — https://go.dev/dl/

The framework depends on `github.com/krewire/libs` (fetched via its git URL); no local checkout is required for `go build`.

### Building

```bash
go build ./...
```

### Testing

```bash
go test ./...
gofmt -l .   # must be empty
go vet ./...
```

### Running the example

```bash
go run ./examples/greet hello --name Alice
GREET_GREETING=Halo go run ./examples/greet hello --name Alice
```

## Specifications

For generic use the framework requires no specs — just `go get` and import `github.com/krewire/framework/*`.

For Krewire ecosystem development, requirements live in `docs/specs/` with `KWF-*` IDs (spec-driven is opt-in). Start with the unified vision `KWF-M8K2Q`, then per-area specs:

- `KWF-M07QS` Web framework, `KWF-PT8OD` SSG, `KWF-0Z671` UI, `KWF-C4087` App
- `KWF-WASM-T4X9P` WASM client runtime
- `KWF-INFRA-B7N3D` Cloud provider abstraction
- `KWF-SVC-L5H2F` Microservice & worker patterns

Testing helpers live in `framework/test` — usable with or without spec tagging (`ftest.Spec` is optional, helpers like `Equal`, `Contains`, `NewRequest` are generic).

## Contributing

Contributions are welcome. Run `gofmt` and `go vet` before submitting, ensure all tests pass. For Krewire ecosystem changes, follow the spec-driven workflow in [`AGENTS.md`](../AGENTS.md) (optional for generic users).

## License

MIT — see [LICENSE](LICENSE).
