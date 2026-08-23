# Specification — Cloud Provider Abstraction

| Field  | Value                                  |
| ------ | -------------------------------------- |
| SpecID | KWF-B7N3D                              |
| Title  | Cloud Provider Abstraction — Multi-Cloud Library-First IaC |
| Status | Draft                                  |
| Date   | 2026-08-21                             |
| Author | Krewire Contributors                    |
| Domain | Framework — Cloud Infrastructure       |

## 1. Context

Krewire currently builds artifacts (`krewire build` for `site`/`book`, `go build`
for `app`/`cli`) but leaves provisioning and deployment to the operator:
`gh-pages`, manual `kubectl apply`, or point-and-click consoles. The unified
framework (KWF-M8K2Q G3) requires Krewire to own the path from "declare
infrastructure in Go/config" to "running in a cloud", the way Pulumi does for
its languages — as a library-first, plan/apply workflow with state and locking.

v1 targets two first providers — AWS and Kubernetes — behind a multi-cloud
abstraction, with preview environments for pull requests and secrets
always referenced, never committed.

## 2. Problem Statement

- Every Krewire site or service needs a separate IaC tool or script to go live;
  there is no standard deploy story from `krewire.yaml`.
- Teams hand-roll deployment per environment, so staging and production drift.
- Preview environments (PR → ephemeral URL) require plumbing that no Krewire
  primitive supports.
- State and locking are ad hoc; concurrent deploys can corrupt infrastructure.

## 3. Goals

- G1 — A `Provider` interface with resource CRUD and `Plan` (desired →
  actionable diff), implemented for AWS and Kubernetes first.
- G2 — Typed resource schemas (Go structs with validation tags) for common
  primitives: compute, database, storage, network, DNS, certificate, secret.
- G3 — State backends (local file, S3/GCS with locking via DynamoDB/Consul)
  and a CLI-managed lifecycle (`krewire deploy --plan/--auto-approve/--destroy`).
- G4 — Preview environments: `krewire deploy --preview` provisions an
  isolated stack per PR, comments the URL, and tears down on merge/close.
- G5 — Secrets always resolved from an external authority (env, AWS Secrets
  Manager) at deploy/run time.

## 4. Non-Goals

- NG1 — Full Terraform parity: import of existing stacks, complex graph
  interpolation, provider ecosystem. v1 covers the resources needed to host
  Krewire workloads.
- NG2 — Drift detection and auto-repair; v1 reports drift on `plan`, does not
  reconcile without an explicit apply.
- NG3 — GCP/Azure providers in v1; the provider contract is designed so they
  can be added without breaking AWS/Kubernetes consumers.
- NG4 — A PaaS control plane; Krewire remains a CLI/library, not a hosted
  service with dashboards and billing.

## 5. Requirements

### 5.1 Provider Contract

| ID            | Requirement                                                            | Priority |
| ------------- | ---------------------------------------------------------------------- | -------- |
| FRK-INFRA-001 | `infra/provider.Provider` exposes `Name() string`, `Create/Read/      | Must     |
|               | `Update/Delete(ctx, Resource) error`, and `Plan(ctx, desired) (Plan,  |          |
|               | `error)`.                                                              |          |
| FRK-INFRA-002 | `Plan` returns an ordered list of `Action{Op, Resource, Reason}` where| Must     |
|               | `Op ∈ {Create, Update, Delete, NoOp}` and dependencies are topologically|         |
|               | sorted per the `DependsOn` graph.                                      |          |
| FRK-INFRA-003 | Providers declare `Resources() []ResourceSchema` for validation and     | Must     |
|               | CLI help; unknown resource kinds error with a did-you-mean suggestion.  |          |
| FRK-INFRA-004 | `Resource` carries `{Kind, ID, Properties map[string]any, DependsOn[]}`| Must     |
|               | and round-trips through JSON state without provider-specific codecs.    |          |

### 5.2 Resource Schema & Validation

| ID            | Requirement                                                            | Priority |
| ------------- | ---------------------------------------------------------------------- | -------- |
| FRK-INFRA-010 | `infra/schema` maps Go structs (tags `validate:"required"`) to JSON    | Must     |
|               | schema via `libs/validate`; schema is used both at decode and at `Plan`.|         |
| FRK-INFRA-011 | Common kinds share a canonical schema: `Compute`, `Database`,           | Must     |
|               | `Storage`, `Network`, `DNS`, `Certificate`, `SecretRef`.                |          |
| FRK-INFRA-012 | Provider-specific fields live under a `provider:` namespace and are     | Must     |
|               | validated by the provider's own schema, never by the core.              |          |

### 5.3 State, Locking, and Plan/Apply

| ID            | Requirement                                                            | Priority |
| ------------- | ---------------------------------------------------------------------- | -------- |
| FRK-INFRA-020 | State is a JSON file `.krewire/state.json` for local, otherwise the     | Must     |
|               | backend configured in `infra.state { backend, bucket, key }`.           |          |
| FRK-INFRA-021 | Remote backends — AWS S3, GCS — acquire a lock before any mutation     | Must     |
|               | (DynamoDB for S3, Consul/etcd for GCS); stale locks expire with a TTL. |          |
| FRK-INFRA-022 | `Plan` is pure and side-effect free; `Apply` applies the plan in      | Must     |
|               | dependency order, persisting state after each resource and rolling      |          |
|               | forward on partial failure (no silent partial state).                   |          |
| FRK-INFRA-023 | `state.Lock()` and `state.Unlock()` are idempotent and return typed    | Should   |
|               | errors (`ErrAlreadyLocked{Owner}`) surfaced by the CLI.                |          |

### 5.4 AWS Provider — First Implementation

| ID            | Requirement                                                            | Priority |
| ------------- | ---------------------------------------------------------------------- | -------- |
| FRK-INFRA-030 | `Compute` maps to ECS/Fargate Service (default) and Lambda (when       | Must     |
|               | `compute.runtime: lambda`); image + env + scaling are CRUD-capable.    |          |
| FRK-INFRA-031 | `Database` maps to RDS Postgres/MySQL and DynamoDB table; credentials  | Must     |
|               | are secret refs, never literal.                                         |          |
| FRK-INFRA-032 | `Storage` is S3 bucket + optional `Storage.Transfer` → CloudFront      | Must     |
|               | origin + distribution; invalidation is part of `Update`.                |          |
| FRK-INFRA-033 | `Network` creates VPC/subnet/security group and `LoadBalancer` as       | Should   |
|               | separate resources with explicit `DependsOn`.                           |          |
| FRK-INFRA-034 | `DNS` creates Route53 record, `Certificate` requests ACM cert +        | Should   |
|               | validation record, `SecretRef` reads AWS Secrets Manager by ARN.       |          |

### 5.5 Kubernetes Provider — First Implementation

| ID            | Requirement                                                            | Priority |
| ------------- | ---------------------------------------------------------------------- | -------- |
| FRK-INFRA-040 | K8s resources are Go structs that render to manifests; `Apply` uses    | Must     |
|               | either `client-go` or `kubectl` fallback, selected at `krewire deploy`  |          |
|               | time.                                                                   |          |
| FRK-INFRA-041 | Supported kinds: `Deployment`, `Service`, `Ingress`, `ConfigMap`,      | Must     |
|               | `Secret`, `HorizontalPodAutoscaler`.                                    |          |
| FRK-INFRA-042 | `Namespace` is a resource; `DependsOn` enforces namespace-before-child | Must     |
|               | ordering and `Delete` cascades via foreground propagation.              |          |

### 5.6 CLI — `krewire deploy`

| ID            | Requirement                                                            | Priority |
| ------------- | ---------------------------------------------------------------------- | -------- |
| FRK-INFRA-050 | `krewire deploy` resolves kind → `build` artifacts → `infra.Plan` →    | Must     |
|               | confirmation → `infra.Apply` → prints endpoints/URLs.                  |          |
| FRK-INFRA-051 | Flags: `--plan` (dry run), `--auto-approve`, `--destroy`, `--env`,    | Must     |
|               | `--preview` (PR-scoped); all return exit code 2 on usage error.       |          |
| FRK-INFRA-052 | `deploy --preview` provisions a namespaced stack (`pr-<number>`),      | Must     |
|               | annotates it with `krewire.io/preview: <pr>` for GC on close.           |          |
| FRK-INFRA-053 | Destroy is ordered reverse-dependency and removes state only after       | Must     |
|               | provider confirms deletion; `--force` is not required.                  |          |

### 5.7 Secrets

| ID            | Requirement                                                            | Priority |
| ------------- | ---------------------------------------------------------------------- | -------- |
| FRK-INFRA-060 | Any field of type `secret.Ref` is resolved at `Plan`/`Apply` time from | Must     |
|               | `env:` or the configured secrets manager; literal secrets are rejected  |          |
|               | by validation.                                                          |          |
| FRK-INFRA-061 | State stores only secret identifiers, never secret values.              | Must     |

## 6. Non-Functional Requirements

- NFR1 — **Library-first**: providers are importable packages consumers can
  wire directly in tests; the CLI is a thin wrapper over the same code.
- NFR2 — **Idempotence**: repeating `Plan` with no desired change yields
  zero `Create/Update/Delete` actions.
- NFR3 — **Observability**: every provider call emits structured logs via
  `log/slog` with `trace_id` when OTel is configured (KWF-L5H2F).
- NFR4 — **Quality gates**: `gofmt`, `go vet ./...`, `go test ./...` in
  affected repos; AWS/K8s tests use fakes or `envtest`, not live accounts.

## 7. Success Criteria

- S1 — `krewire deploy --plan` on a fixture with `Storage` + `Compute`
  prints an ordered plan against the selected provider without mutating state.
- S2 — A static site fixture deploys to an S3 + CloudFront stack and serves
  the SSG output at the printed URL; `krewire deploy --destroy` removes it.
- S3 — A service fixture deploys to a `kind` cluster (CI fixture) and the
  demo `/health` endpoint becomes reachable post-apply.
- S4 — Preview lifecycle (create → comment URL → destroy) completes in CI
  against the local/file backend without a cloud account.

## 8. Related Specifications

| SpecID    | Title                                               |
| --------- | --------------------------------------------------- |
| [KWF-M8K2Q](./KWF-ARCH-M8K2Q-unified-framework-vision.md) | Unified framework vision (parent) |
| [KWF-L5H2F](./KWF-SVC-L5H2F-microservice-patterns.md)    | Microservice & worker patterns (consumer of infra) |
| [KWF-CCI0N](./KWF-STRUCT-CCI0N-app-directory-structure.md) | Project directory structure |
| [KWL-2X1QZ](https://github.com/krewire/libs)         | libs — configuration & validation |

## 9. References

- Pulumi program model: https://www.pulumi.com/docs/concepts/
- Terraform state & locking: https://developer.hashicorp.com/terraform/language/state/locking
- Kubernetes API conventions: https://kubernetes.io/docs/reference/using-api/api-concepts/
