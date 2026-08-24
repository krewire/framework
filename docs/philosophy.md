# Philosophy — Krewire Framework

## Philosophy

**One framework, every workload.** The same `krewire.yaml` and `kiw build` that produce a CLI binary also produce a `site/`, a `book`, a WASM frontend, a worker binary, and an infra plan. The framework does not force a choice between monolith and microservice — it starts as a modular monolith (`KWF-5ZHQV`) and extracts via `service`/`worker`/`infra` only when needed.

**Principles:**

- **Opt-in batteries.** Monolith imports `app`; distributed patterns activate only for `worker`/`service`/`infra` kinds. Binary size unchanged when unused.
- **Stdlib-first & idiomatic Go.** `(value, error)`, functional options, zero-value usability, clear `go doc`.
- **Single config, single CLI.** `krewire.yaml` only; no `ssg.yaml`.
- **One control plane.** `libs/core` is the declarative center (business rules, workload registry); `libs/kern` the imperative center (Kernel/Executor); the framework composes via `kern`.
- **Spec-driven (ecosystem opt-in).** Inside the Krewire ecosystem every feature has a `KWF-*` spec with `FRK-*` rows and `// Tests for <SpecID>` per `KWL-TEST-P8M4L`. The framework itself is generic — usable without spec-driven; helpers in `framework/test` work with or without `Spec()` tagging.
- **Modular at every Scope (SRP/SoC).** Industry **Single Responsibility Principle** (SOLID), **Separation of Concerns**, **High Cohesion/Low Coupling** (Parnas/Constantine). One `Package`/`Func` = one reason to change; no God Module. Even `Func` (`ui.Button`, `framework/tui.Help`) is a module. Maps to `KWL-ARCH-J2K9Q` `Module→Package→Func`.
- **Dogfooded & safe.** Built on Go memory safety, `gofmt`/`go vet`/`go test` gates; `unsafe` never used.


## Contribution

- The framework is generic — no spec required to use `tui`, `web`, `ui`, `app`. For Krewire ecosystem contributions, read [`project-vision.md`](https://github.com/krewire/internal/blob/main/docs/project-vision.md) and `docs/specs/index.md` before changing behavior (spec-driven is opt-in for the ecosystem).
- Add/update tests matching project patterns; keep suite green (`go test ./...`).
- Update `README.md` / `docs/` when public behavior changes; specs are optional for generic users.
