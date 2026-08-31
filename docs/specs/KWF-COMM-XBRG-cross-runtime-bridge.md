# Specification — Cross-Runtime Bridge Protocol

| Field       | Value                                      |
| ----------- | ------------------------------------------ |
| SpecID      | KWF-COMM-XBRG                              |
| Title       | Cross-Runtime Bridge Protocol              |
| Status      | Draft                                      |
| Date        | 2026-08-30                                 |
| Author      | Krewire Contributors                        |
| Domain      | Framework — Communication                 |
| Scope       | COMM                                        |

## Scope

This spec targets the **Krewire ecosystem** at the **Framework** and **Service** scope levels. It defines the communication protocols that allow Krewire Go services to interoperate with external runtimes — specifically Node.js and PHP/Laravel — without shared memory, shared runtime, or shared dependency graph.

Containment: `Workspace ⊃ Module ⊃ Domain ⊃ Service ⊃ Unit`. The bridge protocol operates at the **Service** level (cross-runtime service communication) and the **Domain** level (protocol contracts between Go and external runtimes).

---

## 1. Context

### 1.1 The Multi-Runtime Reality

Modern web applications rarely run on a single runtime. A typical architecture involves:
- **Node.js** for the web layer (frontend rendering, API gateway, real-time WebSocket)
- **PHP** for the application layer (Laravel business logic, ORM, authentication)
- **Go** for performance-critical services (workers, microservices, infrastructure)

These runtimes do not share a process space, a memory heap, or a dependency resolution system. Communication between them must happen through defined, serializable boundaries.

### 1.2 Existing Patterns

The industry has established several patterns for cross-runtime communication:

| Pattern | Mechanism | Latency | Complexity |
|---------|-----------|---------|------------|
| **Process bridge** | Spawn binary, stdin/stdout JSON | ~50-150ms | Low |
| **HTTP/JSON** | REST/gRPC over TCP | ~5-20ms | Low |
| **gRPC** | Protocol Buffers over HTTP/2 | ~1-5ms | Medium |
| **Message queue** | Redis, RabbitMQ, NATS | ~5-50ms | Medium |
| **Shared database** | Both runtimes read/write same DB | Variable | Low |

The `govel` package (`mpge/govel`) demonstrated the process bridge pattern for Go+Laravel: Go binaries read JSON from stdin, write JSON to stdout, and are spawned by PHP's Symfony Process component.

### 1.3 Krewire's Position

Krewire's Go services must communicate with external runtimes using standard, well-documented protocols. This spec defines the canonical communication patterns that Krewire services adopt, ensuring interoperability with any runtime (Node.js, PHP, Python, etc.) without requiring runtime-specific adapters in the Krewire framework.

---

## 2. Problem Statement

### 2.1 The Protocol Gap

When a Krewire Go service needs to communicate with a Node.js or PHP application, there is no standardized protocol defined in the Krewire ecosystem. Each integration either:
- Invent its own ad-hoc protocol (inconsistent, hard to maintain)
- Rely on HTTP alone (no structured error handling, no streaming, no typed contracts)
- Use a runtime-specific SDK (locks the integration to one language)

### 2.2 The Serialization Mismatch

Go uses structs and JSON, Node.js uses objects and JSON, PHP uses arrays and JSON. While JSON is common to all three, the structure, naming conventions, and error handling differ. Without a standardized message format, every cross-runtime integration needs custom serialization logic.

### 2.3 The Lifecycle Mismatch

Go binaries have a distinct lifecycle: compile → spawn → process → exit. Node.js processes are persistent (Event Loop). PHP processes are per-request (FPM). These lifecycle differences affect how connections are managed, how timeouts are handled, and how errors propagate.

---

## 3. Goals

| ID        | Goal                                                                                             | Priority |
| --------- | ------------------------------------------------------------------------------------------------ | -------- |
| G1        | **Standardized message format.** Define a canonical JSON message envelope that works across Go, Node.js, and PHP. | Must |
| G2        | **Process bridge protocol.** Define the stdin/stdout JSON protocol for spawned Go binaries. | Must |
| G3        | **HTTP/gRPC service protocol.** Define how Krewire services expose HTTP/JSON and gRPC endpoints callable from any runtime. | Must |
| G4        | **Message queue protocol.** Define how Krewire workers consume from queues that any runtime publishes to. | Must |
| G5        | **Error envelope.** Define a standardized error format that propagates across runtime boundaries. | Must |
| G6        | **Distributed tracing.** Define how trace IDs propagate across runtime boundaries. | Should |
| G7        | **Streaming support.** Define how streaming data (real-time events, file streams) crosses runtime boundaries. | Could |

---

## 4. Non-Goals

| ID        | Non-Goal                                                                                         |
| --------- | ------------------------------------------------------------------------------------------------ |
| NG1       | Implementing the bridge in a specific language. This spec defines protocols, not language-specific SDKs. |
| NG2       | Replacing any runtime's native communication mechanism. Krewire adopts existing standards (HTTP, gRPC, JSON, Redis). |
| NG3       | Providing a universal RPC framework. Each runtime's native RPC (Go gRPC, Node.js gRPC, PHP gRPC) is used directly. |
| NG4       | Shared in-memory caching between runtimes. Communication is always serialized. |
| NG5       | Protocol buffers as the only serialization format. JSON is the default; Protocol Buffers is an optimization for gRPC. |

---

## 5. Requirements

### 5.1 Message Envelope (Must)

| ID        | Requirement                                                                                      | Priority |
| --------- | ------------------------------------------------------------------------------------------------ | -------- |
| FRK-COMM-001 | Every cross-runtime message uses a canonical envelope with fields: `id` (string, UUID), `type` (string, e.g. `"request"`, `"response"`, `"error"`, `"event"`), `payload` (any), `timestamp` (ISO 8601), `trace_id` (string, optional). | Must |
| FRK-COMM-002 | The envelope is serialized as JSON with UTF-8 encoding. All fields are UTF-8 compatible. | Must |
| FRK-COMM-003 | The `payload` field carries the application-specific data. Its structure is defined per use case, not by the protocol. | Must |
| FRK-COMM-004 | The `id` field enables request-response correlation across runtime boundaries. | Must |
| FRK-COMM-005 | The `trace_id` field propagates distributed trace identifiers from the calling runtime to the called runtime. | Should |

### 5.2 Process Bridge Protocol (Must)

| ID        | Requirement                                                                                      | Priority |
| --------- | ------------------------------------------------------------------------------------------------ | -------- |
| FRK-COMM-010 | A spawned Go binary reads a single JSON envelope from stdin, processes it, and writes a single JSON envelope to stdout. | Must |
| FRK-COMM-011 | The Go binary exits with code 0 on success, code 1 on application error, and code 2 on usage error. | Must |
| FRK-COMM-012 | Error details are written to stderr in the same envelope format (type `"error"`). | Must |
| FRK-COMM-013 | The calling runtime (Node.js `child_process`, PHP `Symfony Process`, etc.) must not pass additional arguments beyond the binary path. All data flows through stdin/stdout. | Must |
| FRK-COMM-014 | Timeout is enforced by the calling runtime (not the Go binary). Default timeout: 30 seconds, configurable. | Must |
| FRK-COMM-015 | The Go binary must not read from stdin after writing the response. Half-duplex communication only. | Must |

### 5.3 HTTP/JSON Service Protocol (Must)

| ID        | Requirement                                                                                      | Priority |
| --------- | ------------------------------------------------------------------------------------------------ | -------- |
| FRK-COMM-020 | Krewire `service` kind exposes HTTP/JSON endpoints at a configurable address (default `:8080`). | Must |
| FRK-COMM-021 | Requests and responses use the canonical message envelope. The envelope is the body of the HTTP request/response. | Must |
| FRK-COMM-022 | Content-Type header is `application/json`. | Must |
| FRK-COMM-023 | Error responses use HTTP status codes (400, 404, 500, etc.) with the envelope `type` set to `"error"`. | Must |
| FRK-COMM-024 | Health check endpoint at `/health` returns `200 OK` with a minimal envelope. | Must |
| FRK-COMM-025 | Krewire services support HTTP/2 for gRPC compatibility. | Should |

### 5.4 gRPC Service Protocol (Should)

| ID        | Requirement                                                                                      | Priority |
| --------- | ------------------------------------------------------------------------------------------------ | -------- |
| FRK-COMM-030 | Krewire `service` kind optionally exposes a gRPC endpoint. The proto definition uses the canonical envelope schema. | Should |
| FRK-COMM-031 | gRPC services support unary, server-streaming, and client-streaming calls. | Should |
| FRK-COMM-032 | Node.js gRPC client and PHP gRPC client can both call the Krewire gRPC service without modification to the Go server. | Must |

### 5.5 Message Queue Protocol (Must)

| ID        | Requirement                                                                                      | Priority |
| --------- | ------------------------------------------------------------------------------------------------ | -------- |
| FRK-COMM-040 | Krewire `worker` kind consumes messages from Redis, RabbitMQ, or NATS. | Must |
| FRK-COMM-041 | The queue message body is the canonical message envelope. | Must |
| FRK-COMM-042 | Any runtime (Node.js, PHP, Go) can publish a message to the queue using the same envelope format. | Must |
| FRK-COMM-043 | Krewire workers acknowledge messages after successful processing. Failed messages are retried or sent to a DLQ. | Must |
| FRK-COMM-044 | Queue connection details (host, port, credentials) are configured in `krewire.yaml`, not hardcoded. | Must |

### 5.6 Error Envelope (Must)

| ID        | Requirement                                                                                      | Priority |
| --------- | ------------------------------------------------------------------------------------------------ | -------- |
| FRK-COMM-050 | Error envelopes include: `id` (correlation), `type` (`"error"`), `error` object with `code` (string), `message` (string), `details` (any, optional), `stack` (string, optional — only in development). | Must |
| FRK-COMM-051 | Error codes are hierarchical: `domain.module.error` (e.g., `worker.process.timeout`, `service.auth.invalid_token`). | Must |
| FRK-COMM-052 | Error messages are always in English. | Must |

### 5.7 Distributed Tracing (Should)

| ID        | Requirement                                                                                      | Priority |
| --------- | ------------------------------------------------------------------------------------------------ | -------- |
| FRK-COMM-060 | The `trace_id` field in the envelope propagates across runtime boundaries. | Should |
| FRK-COMM-061 | Krewire Go services extract `trace_id` from incoming envelopes and include it in outgoing envelopes. | Should |
| FRK-COMM-062 | Trace IDs follow the W3C Trace Context format (`00-{trace_id}-{parent_id}-{flags}`). | Should |

### 5.8 Streaming (Could)

| ID        | Requirement                                                                                      | Priority |
| --------- | ------------------------------------------------------------------------------------------------ | -------- |
| FRK-COMM-070 | For streaming data, the envelope supports a `stream` field with `sequence` (int), `total` (int or null), and `data` (any). | Could |
| FRK-COMM-071 | Streaming over HTTP uses chunked transfer encoding or Server-Sent Events (SSE). | Could |
| FRK-COMM-072 | Streaming over gRPC uses server-streaming or client-streaming RPCs. | Could |

---

## 6. Non-Functional Requirements

| ID | Requirement | Detail |
| -- | ----------- | ------ |
| NFR1 | **Serialization.** JSON encoding/decoding must use `encoding/json` (stdlib). No third-party JSON libraries. |
| NFR2 | **Performance.** Process bridge invocation overhead must not exceed 100ms (excluding Go compilation time). |
| NFR3 | **Memory safety.** The `unsafe` package must not be used in the bridge protocol implementation. |
| NFR4 | **Compatibility.** The protocol must work with Go 1.22+, Node.js 18+, PHP 8.1+. |
| NFR5 | **Testability.** All protocol implementations must have unit tests covering the envelope format, error handling, and timeout behavior. |
| NFR6 | **Observability.** Every cross-runtime call must log the `id` and `trace_id` for debugging. |

---

## 7. Success Criteria

| ID | Criterion | Verification |
| -- | --------- | ------------ |
| S1 | A Go binary spawned by Node.js `child_process` receives a JSON envelope via stdin and returns a JSON envelope via stdout. | Integration test: Node.js → Go binary → Node.js |
| S2 | A Go binary spawned by PHP `Symfony Process` receives a JSON envelope via stdin and returns a JSON envelope via stdout. | Integration test: PHP → Go binary → PHP |
| S3 | A Node.js HTTP client can call a Krewire service and receive a canonical envelope response. | Integration test: `fetch` → Krewire service → `fetch` |
| S4 | A PHP application can publish to Redis and a Krewire worker can consume the message. | Integration test: PHP → Redis → Go worker |
| S5 | Error envelopes propagate correctly across all four communication patterns. | Test: each pattern returns standardized error envelope |
| S6 | `gofmt -l .`, `go vet ./...`, `go test ./...` pass for the bridge protocol implementation. | Quality gate |

---

## 8. Related Specifications

| SpecID | Title | Relationship |
|--------|-------|-------------|
| [KWF-M8K2Q](https://github.com/krewire/framework/blob/main/docs/specs/KWF-ARCH-M8K2Q-unified-framework-vision.md) | Unified Vision | Parent vision. Communication protocols enable the "services + workers + infra" stage. |
| [KWF-SVC-L5H2F](https://github.com/krewire/framework/blob/main/docs/specs/KWF-SVC-L5H2F-microservice-patterns.md) | Microservice & Worker Patterns | Krewire services and workers use this bridge protocol for cross-runtime communication. |
| [KWF-ARCH-P7L2Q](https://github.com/krewire/framework/blob/main/docs/specs/KWF-ARCH-P7L2Q-progressive-pipeline.md) | Progressive Pipeline | Cross-runtime bridges enable the "headless split" stage (Node.js frontend + Go backend). |
| [KWF-T4X9P](https://github.com/krewire/framework/blob/main/docs/specs/KWF-T4X9P-wasm-client-runtime.md) | WASM Client Runtime | WASM components may need to communicate with Go services via the same protocols. |
| `mpge/govel` | Govel | Reference implementation for the process bridge pattern with Laravel. |

---

## 9. References

- [W3C Trace Context](https://www.w3.org/TR/trace-context/) — distributed tracing standard
- [JSON-RPC 2.0](https://www.jsonrpc.org/specification) — message envelope pattern inspiration
- [govel](https://github.com/mpge/govel) — reference implementation for Go+PHP process bridge
- [Go `encoding/json`](https://pkg.go.dev/encoding/json) — stdlib JSON serialization
- [Symfony Process](https://symfony.com/doc/current/components/process.html) — PHP process component
- [Node.js `child_process`](https://nodejs.org/api/child_process.html) — Node.js process module

---

## 10. Revision History

Revision history is tracked by **git**, not in-file metadata.

Initial draft: 2026-08-30.
