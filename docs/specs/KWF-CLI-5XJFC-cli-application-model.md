# Specification — CLI Application Model

| Field       | Value                                       |
| ----------- | ------------------------------------------- |
| SpecID      | KWF-5XJFC                              |
| Title       | CLI Application Model                       |
| Status      | Draft                                       |
| Date        | 2026-08-18                                  |
| Author      | Krewire Contributors                         |
| Domain      | Frameworks — CLI                            |

## 1. Context

The `cli` package is the heart of the Krewire CLI ecosystem: a framework-level
application model built on the Go standard library, integrating argument
parsing (`flag`), logging (`log/slog`), terminal rendering (`term`), and the
shared error/exit-code model (`core`).

This specification defines the application model: `App`, `Command`, dispatch,
and lifecycle.

## 2. Problem Statement

CLI applications in Go are routinely assembled ad hoc: each project hand-writes
argument loops, subcommand dispatch, and exit-code plumbing. The result is
non-canonical behavior (unknown commands and usage errors resolve to arbitrary
codes), duplicated boilerplate, and logic that is awkward to test in isolation.

## 3. Goals

- G1 — Model CLI applications as `App` + `Command` composition.
- G2 — Support nested command/subcommand hierarchies.
- G3 — Delegate argument parsing entirely to the standard library `flag`.
- G4 — Guarantee that every execution path yields a canonical exit code.

## 4. Non-Goals

- NG1 — A bespoke argument parser (stdlib `flag` is mandated).
- NG2 — A plugin/extension runtime in the initial phase.
- NG3 — Async execution scheduling (deferred to the async phase).

## 5. Requirements

### 5.1 Application & Command Model

| ID         | Requirement                                                       | Priority |
| ---------- | ----------------------------------------------------------------- | -------- |
| FRK-AM-001 | Provide an `App` with a name, version, and known commands.        | Must     |
| FRK-AM-002 | Provide a `Command` with a name, description, flag registration, and handler. | Must |
| FRK-AM-003 | Support registering commands and returning the `App` for chaining. | Must     |
| FRK-AM-004 | Support nested command hierarchies (sub-commands under a parent). | Must     |

### 5.2 Dispatch & Execution

| ID         | Requirement                                                       | Priority |
| ---------- | ----------------------------------------------------------------- | -------- |
| FRK-AM-005 | Dispatch on the first token; unknown commands yield the usage exit code. | Must |
| FRK-AM-006 | Parse remaining tokens with `flag.FlagSet` per command; parse failures yield the usage exit code. | Must |
| FRK-AM-007 | Every handler returns an exit code; `Run` returns it to the caller. | Must     |
| FRK-AM-008 | Empty invocation renders help and returns the usage exit code.    | Must     |

### 5.3 Lifecycle & Diagnostics

| ID         | Requirement                                                       | Priority |
| ---------- | ----------------------------------------------------------------- | -------- |
| FRK-AM-009 | Emit structured logs through `log/slog` (dispatch, errors).       | Must     |
| FRK-AM-010 | Keep the `cli` package free of panics across dispatch and parsing. | Must    |
| FRK-AM-011 | Provide the framework meta-package as the single-dependency entry point. | Must |

## 6. Non-Functional Requirements

- NFR1 — **Memory safety.** The `unsafe` package must not be used.
- NFR2 — **Performance.** Dispatch overhead must stay in the microseconds.
- NFR3 — **Portability.** Linux, macOS, and Windows.
- NFR4 — **Quality gates.** `gofmt`, `go vet ./...`, and `go test ./...` must pass.

## 7. Success Criteria

- S1 — Nested commands dispatch correctly and are covered by tests.
- S2 — All non-success paths resolve to `ExitCodeUsage` (2) or `ExitCodeFailure` (1), never raw integers.
- S3 — Package documentation covers every exported identifier.
- S4 — The `greet` example demonstrates the full model end-to-end.

## 8. Related Specifications

| SpecID      | KWF-5XJFC                              |
| --------- | ----------------------------------------------- |
| [KWF-CMBZJ](./KWF-META-CMBZJ-krewire-meta-framework.md) | Krewire Meta-Framework                 |
| [KWF-MFA0T](./KWF-CLI-MFA0T-cli-help-usage.md)         | CLI Help & Usage                     |
| [KWF-KAKQL](./KWF-CLI-KAKQL-cli-errors-diagnostics.md) | CLI Errors & Diagnostics             |
| [KWL-W0J2X](https://github.com/krewire/libs/blob/main/docs/specs/KWL-CORE-W0J2X-errors-exit-codes.md) | Core Errors & Exit Codes |

## 9. References

- [KWL-M1ZKS](https://github.com/krewire/libs/blob/main/docs/specs/KWL-CORE-M1ZKS-krewire-libraries.md) — Krewire Libraries initial specification.
- [KWL-R934Y](https://github.com/krewire/libs/blob/main/docs/specs/KWL-TERM-R934Y-terminal-io-rendering.md) — Terminal I/O & Rendering.