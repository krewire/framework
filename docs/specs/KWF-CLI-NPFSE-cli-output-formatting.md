# Specification — CLI Output & Formatting

| Field       | Value                                       |
| ----------- | ------------------------------------------- |
| SpecID      | KWF-NPFSE                              |
| Title       | CLI Output & Formatting                     |
| Status      | Draft                                       |
| Date        | 2026-08-18                                  |
| Author      | Krewire Contributors                         |
| Domain      | Frameworks — CLI                            |

## 1. Context

Consistent output is essential for composable CLIs (piped to other tools) and
for humans. This specification formalizes how Krewire CLI commands write data:
stream separation, machine-readable modes, color policy, and progress
conventions.

## 2. Problem Statement

CLI tools mix data output with diagnostics, spill colors into
non-TTY/piped destinations, and offer no stable machine-readable mode.
Downstream automation (pipes, scripts, CI) receives polluted, non-deterministic
streams, and human users get noisy, inconsistent output.

## 3. Goals

- G1 — Separate data output (stdout) from diagnostics (stderr).
- G2 — Support deterministic human-readable and machine-readable (JSON) output.
- G3 — Apply a single color policy across the ecosystem (`NO_COLOR`, `TERM=dumb`).
- G4 — Keep structural primitives ready for progress/spinner features.

## 4. Non-Goals

- NG1 — Full terminal UI (TUI) toolkit in the initial phase.
- NG2 — Automatic JSON schema generation.
- NG3 — Localization of output text.

## 5. Requirements

### 5.1 Stream Separation

| ID         | Requirement                                                       | Priority |
| ---------- | ----------------------------------------------------------------- | -------- |
| FRK-OU-001 | Command results are written only to stdout.                       | Must     |
| FRK-OU-002 | Logs, warnings, and errors are written only to stderr.            | Must     |
| FRK-OU-003 | Progress indicators never pollute stdout.                         | Must     |

### 5.2 Output Modes

| ID         | Requirement                                                       | Priority |
| ---------- | ----------------------------------------------------------------- | -------- |
| FRK-OU-004 | Provide a machine-readable (JSON) mode for structured results.    | Should   |
| FRK-OU-005 | Output in a given mode must be deterministic (stable ordering).   | Must     |
| FRK-OU-006 | Human mode must degrade to plain text when colors are unsupported. | Must    |

### 5.3 Progress & Spinner

| ID         | Requirement                                                       | Priority |
| ---------- | ----------------------------------------------------------------- | -------- |
| FRK-OU-007 | Expose rendering primitives for progress/spinners via `term`.     | Should   |
| FRK-OU-008 | Progress must auto-disable when the destination is not a TTY.     | Should   |

## 6. Non-Functional Requirements

- NFR1 — **Memory safety.** The `unsafe` package must not be used.
- NFR2 — **Performance.** Formatting should not buffer unbounded output.
- NFR3 — **Portability.** Linux, macOS, and Windows.
- NFR4 — **Quality gates.** `gofmt`, `go vet ./...`, and `go test ./...` must pass.

## 7. Success Criteria

- S1 — A command piped into another tool receives clean data with no diagnostics.
- S2 — JSON mode output validates and is byte-stable across runs.
- S3 — With `NO_COLOR` set, output contains no escape sequences.

## 8. Related Specifications

| SpecID      | KWF-NPFSE                              |
| --------- | ----------------------------------------------- |
| [KWF-5XJFC](./KWF-CLI-5XJFC-cli-application-model.md) | CLI Application Model                |
| [KWF-MFA0T](./KWF-CLI-MFA0T-cli-help-usage.md)         | CLI Help & Usage                     |
| [KWL-R934Y](https://github.com/krewire/libs/blob/main/docs/specs/KWL-TERM-R934Y-terminal-io-rendering.md) | Terminal I/O & Rendering            |

## 9. References

- [KWF-CMBZJ](./KWF-META-CMBZJ-krewire-meta-framework.md) — Krewire Meta-Framework initial specification.