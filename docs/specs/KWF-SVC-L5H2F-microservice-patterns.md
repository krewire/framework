# Specification — Microservice & Worker Patterns

| Field  | Value                               |
| ------ | ----------------------------------- |
| SpecID | KWF-L5H2F                           |
| Title  | Microservice & Worker Patterns      |
| Status | Draft                               |
| Date   | 2026-08-21                          |
| Author | Krewire Contributors                 |
| Domain | Framework — Distributed Systems     |

## 1. Context

The modular-monolith spec (KWF-5ZHQV) gives teams explicit module boundaries
and an extraction checklist for when a module graduates to an independent
service. Checklist items — shared contracts, a service client, separate data
store, observability, independent deploy — currently have no runtime
counterpart. Teams that follow the checklist must supply discovery, config,
an API gateway, resilience patterns, tracing, messaging, and background-job
handling themselves.

The unified framework (KWF-M8K2Q G4) requires these as opt-in, batteries-
included packages under `framework/service` and `framework/worker`, sharing the
single `krewire.yaml` and not taxing monolith-only projects.

Infrastructure that these patterns target is provided by KWF-B7N3D; the client
runtime (KWF-T4X9P) and theming story are not prerequisites for this spec's
delivery, only its vocabulary of services and endpoints.

## 2. Problem Statement

- Extracting a module into a service requires choosing and wiring a service
  registry, config distribution, gateway, and resilience primitives from
  scratch.
- No standard exists inside Krewire for distributed config hot-reload, tracing
  propagation, or a message bus abstraction.
- Background work (queues, scheduled jobs, retries, dead letters) has no
  framework; teams hand-roll workers incompatible with the app container.
- Monoliths that never extract should not pay any runtime or binary-size cost
  for microservice machinery.

## 3. Goals

- G1 — A `service/registry` with `Register`, `Deregister`, `Discover`, and
  `Watch` backends (Consul, etcd, NATS, DNS).
- G2 — A `service/config` center supporting distributed values with
  push `Watch`; backends include etcd/Consul/S3/Git + local/file fallback.
- G3 — An API gateway (`web/gateway`) with route table, middleware chain,
  rate limiting, auth, and observability hooks.
- G4 — Resilience primitives: circuit breaker, retry with backoff/jitter,
  timeout via `context`, bulkhead/semaphore.
- G5 — Distributed tracing via OpenTelemetry with W3C trace-context
  propagation and pluggable exporters.
- G6 — Message bus abstraction (`Publisher`/`Subscriber`/`Stream`) with a
  NATS JetStream primary and Kafka adapter slot.
- G7 — Worker framework: job interfaces, queues, priority/delay/cron,
  retries, dead-letter queue, and a `krewire worker` runner.

## 4. Non-Goals

- NG1 — A sidecar service mesh (Envoy/Istio); Krewire provides library-level
  patterns, not per-service sidecars or data-plane proxies.
- NG2 — A custom wire protocol or RPC framework; domain contracts stay as Go
  interfaces (KWF-5ZHQV) and the gateway speaks plain HTTP/gRPC.
- NG3 — Kafka optimality: Kafka is a secondary message adapter; the v1 primary
  is NATS JetStream.
- NG4 — Mandatory coupling: a monolith that imports none of these packages
  behaves as before; `project.kind: service` is an opt-in signal.

## 5. Requirements

### 5.1 Service Registry

| ID           | Requirement                                                              | Priority |
| ------------ | ------------------------------------------------------------------------ | -------- |
| FRK-SVC-001  | `service/registry.Registry` exposes `Register(Service) error`,           | Must     |
|              | `Deregister(ID) error`, `Discover(ServiceName) ([]Endpoint, error)`,     |          |
|              | `Watch(ServiceName) (<-chan []Endpoint, Cancel)`.                         |          |
| FRK-SVC-002  | `Service{ID, Name, Addr, Meta, HealthCheckURL}` is the registration     | Must     |
|              | payload; health check failure triggers automatic deregistration policy.   |          |
| FRK-SVC-003  | Backends: Consul, etcd, NATS, DNS. Selection via `service.registry {     | Must     |
|              | `backend: consul|etcd|nats|dns}` in `krewire.yaml`.                        |          |
| FRK-SVC-004  | Registry consumers tolerate transient backend unavailability and do not   | Must     |
|              | panic or busy-loop on `Watch` errors; backoff is mandatory.              |          |

### 5.2 Distributed Configuration

| ID           | Requirement                                                              | Priority |
| ------------ | ------------------------------------------------------------------------ | -------- |
| FRK-SVC-010  | `service/config.Center` exposes `Get(key) (Value, error)`, `Set(key,    | Must     |
|              | `Value) error`, `Watch(prefix) (<-chan Change, Cancel)`.                 |          |
| FRK-SVC-011  | Hot reload: watchers push changes; registered callbacks receive them      | Must     |
|              | atomically and the process does not restart.                              |          |
| FRK-SVC-012  | Backends: etcd, Consul, S3, Git, local/file. Typed decode uses           | Must     |
|              | `libs/config` + `libs/validate` so the same schema validates local and   |          |
|              | remote config.                                                            |          |
| FRK-SVC-013  | Config center composes with `infra/state` for secrets — secret values    | Should   |
|              | are refs resolved at use time, not stored in config center state.        |          |

### 5.3 API Gateway

| ID           | Requirement                                                              | Priority |
| ------------ | ------------------------------------------------------------------------ | -------- |
| FRK-SVC-020  | `web/gateway.Gateway` owns a `Route{Path, Method, Service, Middleware[]  | Must     |
|              | `, RateLimit, Auth}` table and proxies via the registry's `Discover`.   |          |
| FRK-SVC-021  | Built-in middleware: `Logger` (`log/slog`), `Trace` (OTel), `CORS`,     | Must     |
|              | `Auth`, `RateLimit`, `CircuitBreaker` (delegates to resilience package). |          |
| FRK-SVC-022  | Gateway reloads routes atomically from config-center without dropping    | Should   |
|              | in-flight requests.                                                       |          |
| FRK-SVC-023  | Missing upstream returns `502` with a structured `Problem` JSON, not a  | Must     |
|              | HTML error page.                                                          |          |

### 5.4 Resilience

| ID           | Requirement                                                              | Priority |
| ------------ | ------------------------------------------------------------------------ | -------- |
| FRK-SVC-030  | Circuit breaker is a state machine `closed → open → half-open` with     | Must     |
|              | configurable thresholds and event callbacks `OnStateChange`.              |          |
| FRK-SVC-031  | Retry supports exponential backoff + jitter, max attempts, and            | Must     |
|              | `RetryIf(error) bool` predicate; respects caller `context` deadline.     |          |
| FRK-SVC-032  | Timeout is driven by `context.WithTimeout`; no separate timer goroutine  | Must     |
|              | per call after the context expires.                                       |          |
| FRK-SVC-033  | Bulkhead/semaphore bounds concurrent calls per dependency and returns     | Should   |
|              | `ErrBulkheadFull` typed error when tripped.                               |          |

### 5.5 Tracing

| ID           | Requirement                                                              | Priority |
| ------------ | ------------------------------------------------------------------------ | -------- |
| FRK-SVC-040  | `service/tracing` configures OTel SDK from `krewire.yaml service.tracing  | Must     |
|              | `{ exporter, endpoint, sampler }` and provides `Tracer(name)` accessor.  |          |
| FRK-SVC-041  | HTTP client/server middleware propagate W3C `traceparent` and emit spans | Must     |
|              | without manual header handling by application code.                       |          |
| FRK-SVC-042  | Exporters: OTLP (default), Jaeger, Zipkin, stdout; selection is config- | Should   |
|              | driven, not compile-time.                                                 |          |

### 5.6 Messaging

| ID           | Requirement                                                              | Priority |
| ------------ | ------------------------------------------------------------------------ | -------- |
| FRK-SVC-050  | `service/messaging` exports `Publisher{Publish(Subject, []byte)}`,       | Must     |
|              | `Subscriber{Subscribe(Subject, Handler) (Sub, error)}`, `Stream` with     |          |
|              | consumer groups and at-least-once delivery contract.                      |          |
| FRK-SVC-051  | Primary backend is NATS JetStream; Kafka adapter conforms to the same     | Must     |
|              | interface and is tested against the interface contract, not provider-     |          |
|              | specific tests alone.                                                     |          |
| FRK-SVC-052  | Subscribers auto-nack and redeliver on handler error; poison messages    | Must     |
|              | routed to a DLQ via the worker package's queue.                           |          |

### 5.7 Worker Framework

| ID           | Requirement                                                              | Priority |
| ------------ | ------------------------------------------------------------------------ | -------- |
| FRK-SVC-060  | `worker.Job` is `Run(context.Context) error`; queue operations are        | Must     |
|              | `Enqueue(Job, Options{Priority, Delay, Cron})`, `Dequeue`, `Ack`, `Nack`.|          |
| FRK-SVC-061  | `krewire worker` runs workers declared in `worker:` config or registered  | Must     |
|              | in the app container; native `krewire` commands, not an external runner.  |          |
| FRK-SVC-062  | Retries (configurable policy) and dead-letter queue are part of the      | Must     |
|              | queue contract; DLQ inspection is CLI-driven (`krewire worker dlq ...`).  |          |
| FRK-SVC-063  | Backends: NATS, Redis, PostgreSQL (pg_boss-style advisory lock). The     | Must     |
|              | default for local dev is an in-memory backend requiring no external deps. |          |

## 6. Non-Functional Requirements

- NFR1 — **Opt-in cost**: importing only `framework/app` without `service` or
  `worker` leaves binary size and startup time indistinguishable from the
  current monolith baseline.
- NFR2 — **Testability**: every interface has a fake/in-memory implementation
  usable in `go test` without containers or network services.
- NFR3 — **Observability**: registry, gateway, messaging, and workers emit
  `log/slog` structured logs and OTel spans when tracing is enabled.
- NFR4 — **Quality gates**: `gofmt -l .`, `go vet ./...`, `go test ./...` in
  `framework` and `krewire` pass; live-container tests are behind `CGO_ENABLED`
  / build tags and not required for the local gate.

## 7. Success Criteria

- S1 — A three-service demo (gateway + two domain services) starts from
  `krewire dev`, registers, and the gateway's routed request returns `200` and
  emits an OTel trace covering gateway → downstream spans.
- S2 — Killing a downstream service trips the caller's circuit breaker; retry
  and timeout remain within the configured envelope (fixture test).
- S3 — A worker queue fixture enqueues a job with delay + retry policy,
  processes it through the NATS-backed queue, and routes the poison variant
  to the DLQ inspectable via `krewire worker dlq list`.
- S4 — The same app compiled without importing `service`/`worker` is produced
  and still passes its original `go test ./...` and `krewire build` unchanged.

## 8. Related Specifications

| SpecID     | Title                                                        |
| ---------- | ------------------------------------------------------------ |
| [KWF-M8K2Q](./KWF-ARCH-M8K2Q-unified-framework-vision.md)     | Unified framework vision (parent)     |
| [KWF-5ZHQV](./KWF-ARCH-5ZHQV-modular-monolith-architecture.md) | Modular monolith & extraction path    |
| [KWF-B7N3D](./KWF-INFRA-B7N3D-cloud-provider-abstraction.md)   | Cloud infra (target for extracted services) |
| [KWF-C9WLJ](./KWF-DI-C9WLJ-app-container-service-providers.md) | DI container & providers             |
| [KWF-C4087](./KWF-APP-C4087-krewire-app-framework.md)         | App framework (composition root)      |

## 9. References

- "Release It!" — Michael Nygard (resilience patterns)
- OpenTelemetry specification: https://opentelemetry.io/docs/specs/otel/
- NATS JetStream: https://docs.nats.io/nats-concepts/jetstream
- PostgreSQL advisory locks: https://www.postgresql.org/docs/current/explicit-locking.html
