# Specification — CLI Configuration

| Field       | Value                                       |
| ----------- | ------------------------------------------- |
| SpecID      | KWF-FGNZ9                              |
| Title       | CLI Configuration                          |
| Status      | Draft                                       |
| Date        | 2026-08-18                                  |
| Author      | Krewire Contributors                         |
| Domain      | Frameworks — CLI                            |

## 1. Context

Krewire CLI applications source configuration from the environment using Go
standard-library primitives (`os` environment variables, `strconv` coercion).
This specification defines the convention: naming, typing, defaults, and
precedence.

## 2. Problem Statement

Environment-based configuration is handled per-project with no shared
convention: variable names, parsing, defaults, and precedence all differ.
Users hit surprise settings and opaque errors, and maintainers repeat
coercion/validation code that stdlib `os` and `strconv` already cover.

## 3. Goals

- G1 — Standardize environment-variable naming across applications.
- G2 — Provide type-safe coercion with explicit defaults.
- G3 — Establish a single precedence order for flags, environment, and defaults.

## 4. Non-Goals

- NG1 — File-based configuration (TOML/YAML/JSON) in the initial phase.
- NG2 — Remote/secret management or vault integration.
- NG3 — A configuration schema/DTO validation framework.

## 5. Requirements

### 5.1 Naming & Sources

| ID         | Requirement                                                       | Priority |
| ---------- | ----------------------------------------------------------------- | -------- |
| FRK-CF-001 | Load configuration from environment variables via `os`.           | Must     |
| FRK-CF-002 | Use a `<APP>_` prefix (e.g. `GREET_`) with `SCREAMING_SNAKE_CASE` keys. | Must |
| FRK-CF-003 | Expose each setting with a documented default.                    | Must     |

### 5.2 Typing & Validation

| ID         | Requirement                                                       | Priority |
| ---------- | ----------------------------------------------------------------- | -------- |
| FRK-CF-004 | Coerce values to typed settings using `strconv`.                  | Must     |
| FRK-CF-005 | Invalid values surface as usage errors (exit code 2).             | Must     |
| FRK-CF-006 | Missing settings fall back to defaults without error.             | Must     |

### 5.3 Precedence

| ID         | Requirement                                                       | Priority |
| ---------- | ----------------------------------------------------------------- | -------- |
| FRK-CF-007 | Precedence order: flags > environment variables > defaults.       | Must     |

## 6. Non-Functional Requirements

- NFR1 — **Memory safety.** The `unsafe` package must not be used.
- NFR2 — **Simplicity.** No external configuration dependencies.
- NFR3 — **Portability.** Linux, macOS, and Windows.
- NFR4 — **Quality gates.** `gofmt`, `go vet ./...`, and `go test ./...` must pass.

## 7. Success Criteria

- S1 — Coercion and validity rules are covered by tests.
- S2 — The `greet` example demonstrates env-based configuration end-to-end.
- S3 — No framework package imports a non-stdlib configuration library.

## 8. Related Specifications

| SpecID      | KWF-FGNZ9                              |
| --------- | ----------------------------------------------- |
| [KWF-5XJFC](./KWF-CLI-5XJFC-cli-application-model.md) | CLI Application Model                |
| [KWF-KAKQL](./KWF-CLI-KAKQL-cli-errors-diagnostics.md) | CLI Errors & Diagnostics             |
| [KWF-PZ5JU](./KWF-CLI-PZ5JU-cli-scaffolding.md)         | CLI Scaffolding                      |

## 9. References

- [KWF-CMBZJ](./KWF-META-CMBZJ-krewire-meta-framework.md) — Krewire Meta-Framework initial specification.