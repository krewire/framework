# Specification — CLI Errors & Diagnostics

| Field       | Value                                       |
| ----------- | ------------------------------------------- |
| SpecID      | KWF-KAKQL                              |
| Title       | CLI Errors & Diagnostics                    |
| Status      | Draft                                       |
| Date        | 2026-08-18                                  |
| Author      | Krewire Contributors                         |
| Domain      | Frameworks — CLI                            |

## 1. Context

Krewire CLI applications must fail predictably. This specification defines how
errors surface to the user and to the operating system: error formatting,
exit-code propagation, panic containment, and structured diagnostics via
`log/slog`.

## 2. Problem Statement

Error handling across CLIs is inconsistent and non-deterministic: ad-hoc exit
values, vague one-off messages, and uncontained panics that corrupt output
streams. Automation cannot reliably tell misuse (usage) from runtime failure,
and users lack quick, structured diagnostics.

## 3. Goals

- G1 — Every failure state maps to a canonical exit code (0/1/2).
- G2 — Errors are communicated consistently on stderr.
- G3 — Runtime panics are contained and reported without corrupting output.

## 4. Non-Goals

- NG1 — A bespoke logging framework (stdlib `log/slog` is mandated).
- NG2 — Stack-trace-based debugging features in the initial phase.
- NG3 — Remote error reporting/telemetry.

## 5. Requirements

### 5.1 Error Mapping

| ID         | Requirement                                                       | Priority |
| ---------- | ----------------------------------------------------------------- | -------- |
| FRK-ER-001 | Map point-of-failure errors to `ExitCodeFailure` (1).             | Must     |
| FRK-ER-002 | Map invalid inputs (arguments/configuration) to `ExitCodeUsage` (2). | Must  |
| FRK-ER-003 | Success paths return `ExitCodeSuccess` (0) only.                  | Must     |
| FRK-ER-004 | Reuse the `core` error model; no raw-integer exit codes in handlers. | Must  |

### 5.2 Diagnostics

| ID         | Requirement                                                       | Priority |
| ---------- | ----------------------------------------------------------------- | -------- |
| FRK-ER-005 | Print `"<app>: <message>"`-style errors to stderr.                | Must     |
| FRK-ER-006 | Emit structured log records for failures via `log/slog`.          | Must     |
| FRK-ER-007 | Recover from panics at the dispatch boundary, exiting with `Failure`. | Must  |
| FRK-ER-008 | Never interleave diagnostics into stdout data streams.            | Must     |

## 6. Non-Functional Requirements

- NFR1 — **Memory safety.** The `unsafe` package must not be used.
- NFR2 — **Determinism.** Identical failure inputs produce identical diagnostics.
- NFR3 — **Portability.** Linux, macOS, and Windows.
- NFR4 — **Quality gates.** `gofmt`, `go vet ./...`, and `go test ./...` must pass.

## 7. Success Criteria

- S1 — All CLI tests assert canonical exit codes, never ad-hoc integers.
- S2 — A panicking handler is contained and yields exit code 1.
- S3 — Diagnostics never appear on stdout.

## 8. Related Specifications

| SpecID      | KWF-KAKQL                              |
| --------- | ----------------------------------------------- |
| [KWF-5XJFC](./KWF-CLI-5XJFC-cli-application-model.md) | CLI Application Model                |
| [KWF-NPFSE](./KWF-CLI-NPFSE-cli-output-formatting.md) | CLI Output & Formatting             |
| [KWL-W0J2X](https://github.com/krewire/libs/blob/main/docs/specs/KWL-CORE-W0J2X-errors-exit-codes.md) | Core Errors & Exit Codes            |

## 9. References

- [KWF-CMBZJ](./KWF-META-CMBZJ-krewire-meta-framework.md) — Krewire Meta-Framework initial specification.