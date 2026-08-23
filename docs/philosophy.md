# Philosophy — Krewire Framework

## Philosophy

**One framework, every workload.** The same `krewire.yaml` and `krewire build` that produce a CLI binary also produce a `site/`, a `book`, a WASM frontend, a worker binary, and an infra plan. The framework does not force a choice between monolith and microservice — it starts as a modular monolith (`KWF-5ZHQV`) and extracts via `service`/`worker`/`infra` only when needed.

**Principles:**

- **Opt-in batteries.** Monolith imports `app`; distributed patterns activate only for `worker`/`service`/`infra` kinds. Binary size unchanged when unused.
- **Stdlib-first & idiomatic Go.** `(value, error)`, functional options, zero-value usability, clear `go doc`.
- **Single config, single CLI.** `krewire.yaml` only; no `ssg.yaml`.
- **Spec-driven.** Every feature has a `KWF-*` spec with `FRK-*` requirement rows before code; requirements declare `Scope` (`KWL-ARCH-J2K9Q`); tests are `// Tests for <SpecID>` per `KWL-TEST-P8M4L`.
- **Modular at every Scope (SRP/SoC).** Industry **Single Responsibility Principle** (SOLID), **Separation of Concerns**, **High Cohesion/Low Coupling** (Parnas/Constantine). One `Package`/`Func` = one reason to change; no God Module. Even `Func` (`ui.Button`, `framework/tui.Help`) is a module. Maps to `KWL-ARCH-J2K9Q` `Module→Package→Func`.
- **Dogfooded & safe.** Built on Go memory safety, `gofmt`/`go vet`/`go test` gates; `unsafe` never used.


## Contribution

- Read [`project-vision.md`](https://github.com/krewire/internal/blob/main/docs/project-vision.md) and `docs/specs/index.md` before changing behavior.
- Add/update tests matching project patterns; keep suite green.
- Update `README.md` / `docs/` and specs when public behavior changes; follow ecosystem spec conventions.
