# Specification — Krewire Framework

| Field       | Value                                       |
| ----------- | ------------------------------------------- |
| SpecID      | KWF-CMBZJ                              |
| Title       | Krewire Meta-Framework — Initial Specification |
| Status      | Draft                                       |
| Date        | 2026-08-18                                  |
| Author      | Krewire Contributors                         |
| Domain      | Frameworks                                  |

## 1. Context

Krewire is a **meta-framework** written in Go. Unlike a conventional
framework built within a single, self-contained ecosystem, Krewire composes and
orchestrates modules sourced across multiple ecosystems and repositories into
one cohesive, high-performance foundation.

The framework is organized as a Go module monorepo hosting the framework's own
orchestration packages. Shared primitives and utilities are maintained
externally in the **Krewire Libraries** monorepo (`libs`) and consumed as
dependencies; Krewire does not re-implement them.

### 1.1 Current State

- Module root `github.com/krewire/framework` with the `framework/` meta-package as the entry point.
- `tui/` provides the integrated command-line application model.
- The module depends on `github.com/krewire/libs` v0.1.0, fetched from its git URL.
- Shared module root: Go 1.22, MIT license.

## 2. Problem Statement

The Go ecosystem offers CLI authors two unpalatable extremes: build from scratch
with only the standard library, re-implementing dispatch, exit codes, help,
output, and configuration for every project; or adopt a "batteries-included"
framework that couples the application to one vendor's platform and resists
modular adoption.

As a result, commands behave inconsistently across tools, non-canonical exit
codes break automation, and teams duplicate effort that the standard library
already covers. Krewire's meta-framework answers this by composing modular,
ecosystem-agnostic libraries (`libs`) with the Go standard library into
a single cohesive foundation — first-class for the CLI ecosystem and shaped
for expansion into async, HTTP, server, and worker ecosystems.

## 3. Goals

- G1 — Provide a single, cohesive entry point (`framework`) for adopting the Krewire meta-framework.
- G2 — Establish the **CLI ecosystem** as the first-class, supported ecosystem in the initial phase.
- G3 — Design the architecture to permit incremental expansion into larger ecosystems (async runtime, HTTP, server, background workers).
- G4 — Compose modules from external ecosystems without coupling to any single implementation.
- G5 — Maintain production quality: memory safety, documented public APIs, and CI-enforced formatting/vetting.

## 4. Non-Goals

The following are explicitly out of scope for the initial phase:

- NG1 — Re-implementing primitives already provided by `libs` or the Go standard library.
- NG2 — Building a full async runtime or HTTP server from scratch in this phase.
- NG3 — Monolithic "batteries-included-only" design; modular adoption must remain possible.
- NG4 — GUI/desktop application support (deferred to a later phase).
- NG5 — Backwards compatibility guarantees before version `1.0.0`.

## 5. Scope — Initial Phase: CLI Ecosystem

The initial phase focuses on a framework-level CLI solution that composes
reusable building blocks (sourced from `libs` and the Go standard
library) into an integrated developer experience.

### 5.1 Framework-Level Capabilities

| ID          | Requirement                                                            | Priority |
| ----------- | ----------------------------------------------------------------------- | -------- |
| FRK-CLI-001 | Provide a `cli` package exposing the framework's CLI application model. | Must     |
| FRK-CLI-002 | Support command and subcommand hierarchies with explicit definitions.   | Must     |
| FRK-CLI-003 | Integrate argument parsing from the Go standard library (`flag`), not a bespoke parser. | Must |
| FRK-CLI-004 | Provide uniform output and formatting conventions across CLI packages.  | Should   |
| FRK-CLI-005 | Provide consistent error handling, diagnostics, and exit-code semantics. | Must    |
| FRK-CLI-006 | Integrate configuration loading and validation via the Go standard library (`os` environment variables and `strconv`). | Should |
| FRK-CLI-007 | Provide scaffolding (`init`/`new`) for bootstrapping new Krewire CLI projects. | Should |
| FRK-CLI-008 | Expose all framework packages through the `framework` meta-package.     | Must    |

### 5.2 Deliverables

- D1 — `tui/` implementing FRK-CLI-001..007.
- D2 — Updated `framework/` meta-package documenting the composed packages (FRK-CLI-008).
- D3 — Example CLI applications under `examples/` demonstrating usage.
- D4 — CI workflow enforcing `gofmt`, `go vet`, and `go test ./...`.

## 6. Architecture Constraints

- C1 — The module is a Go monorepo; the `framework` meta-package is the public entry point.
- C2 — Dependencies point **one direction only**: framework packages → `libs` packages; cyclic dependencies are prohibited.
- C3 — `libs` packages are referenced by module version (e.g. `github.com/krewire/libs@v0.1.0`) fetched from its git URL; a local `replace` directive may be used during development.
- C4 — Each package is versioned **independently**; the module root defines the shared language version and license.
- C5 — The `unsafe` package must not be used; all exported identifiers must be documented.
- C6 — New capabilities are introduced as additive packages; breaking changes are not allowed in minor/patch releases.

## 7. Non-Functional Requirements

- NFR1 — **Memory safety.** The `unsafe` package must not be used.
- NFR2 — **Performance.** CLI startup and command dispatch overhead must remain minimal.
- NFR3 — **Portability.** All packages must target Linux, macOS, and Windows.
- NFR4 — **Minimum Go version.** The minimum supported Go version must be documented in the README.
- NFR5 — **Quality gates.** `gofmt`, `go vet ./...`, and `go test ./...` must pass in CI.

## 8. Success Criteria

- S1 — A new project can be bootstrapped, built, and run using only the framework packages.
- S2 — Framework packages pass `go vet` with no findings and all tests pass.
- S3 — At least one complete example CLI application ships under `examples/`.
- S4 — Documentation comments (rendered by `go doc`) cover every exported identifier.

## 9. Future Phases (Expansion)

The architecture must accommodate, in order of priority:

| Phase | Ecosystem            | Planned Packages                  |
| ----- | -------------------- | --------------------------------- |
| P1    | CLI                  | `cli`                             |
| P2    | Async                | `async`                           |
| P3    | HTTP                 | `http`, `router`                  |
| P4    | Server               | `server` (async + http + router)  |
| P5    | Workers              | `worker`                          |

Each phase is additive and must not break the previous phases.

## 10. Related Specifications

| SpecID      | KWF-CMBZJ                              |
| --------- | ----------------------------------------------- |
| [KWF-5XJFC](./KWF-CLI-5XJFC-cli-application-model.md) | CLI Application Model                |
| [KWF-MFA0T](./KWF-CLI-MFA0T-cli-help-usage.md)         | CLI Help & Usage                     |
| [KWF-NPFSE](./KWF-CLI-NPFSE-cli-output-formatting.md) | CLI Output & Formatting              |
| [KWF-KAKQL](./KWF-CLI-KAKQL-cli-errors-diagnostics.md) | CLI Errors & Diagnostics             |
| [KWF-FGNZ9](./KWF-CLI-FGNZ9-cli-configuration.md)     | CLI Configuration                    |
| [KWF-PZ5JU](./KWF-CLI-PZ5JU-cli-scaffolding.md)       | CLI Scaffolding                      |

## 11. References

- [Krewire Libraries — KWL-M1ZKS](https://github.com/krewire/libs/blob/main/docs/specs/KWL-CORE-M1ZKS-krewire-libraries.md) — Initial specification for the Krewire Libraries monorepo.
- [libs](https://github.com/krewire/libs) — modular reusable libraries hosting `core` and `term`.
- Project `README.md` for building and testing instructions.