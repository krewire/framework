# Specification — Authentication & Policy Gates

| Field  | Value |
| ------ | ----- |
| SpecID | KWF-WEB-B2X7D |
| Title  | Auth: Basic, JWT (HS256), Policy Gates (before/after) |
| Status | Draft |
| Date   | 2026-08-23 |
| Author | Krewire Contributors |
| Domain | Framework — Web — AuthN/AuthZ |

## 1. Context

Following the expressive HTTP layer (`KWF-WEB-P3V8X`) and security/state stack
(`KWF-WEB-R9T4C`), apps still lack first-class authentication and authorization.
Handlers need a unified notion of *who is calling* (identity) regardless of
credential transport (Basic vs JWT), and routes need declarative gates that run
before handlers (authorize) and hooks that run after them (audit).

Constraints honored: stdlib-only crypto (`crypto/hmac`, `crypto/sha256`,
`crypto/subtle`, `encoding/base64`) — no external JWT dependency; identities and
policies compose as plain middlewares with `Use`/`Group`/route `.Use`.

## 2. Goals

- G1 — `BasicAuth`: RFC-7617 parsing, pluggable verifier, proper `WWW-Authenticate` challenge, `Identity` in context.
- G2 — Minimal correct JWT: HS256 sign/verify with `exp` enforcement, alg pinning (rejects `none`/other algs), Bearer extraction, claims surfaced on the identity.
- G3 — Policy gates: composable `Policy` funcs run before the handler via `Require(...)`; post-handler observers via `AfterRequest`; built-ins `Authenticated()` and `WithRoles(...)`.
- G4 — Named policy registry so route groups declare intent: `web.PolicySet{"admin": ...}.Require("admin")`.

## 3. Non-Goals

- NG1 — No RS256/ES256/JWKS (HMAC shared-secret MVP); no refresh-token machinery.
- NG2 — No login endpoints/user store — verifiers are injected by the app.
- NG3 — No RBAC graph; roles are plain strings on the identity.

## 4. Requirements

### Identity

| ID | Requirement | Priority |
|----|-------------|----------|
| FRK-AUTH-001 | `Identity{ Subject string; Method string; Roles []string; Claims map[string]any }` resolved via `*Request.Identity()` / `IdentityFrom(ctx)` (nil when anonymous). | Must |

### Basic Auth

| ID | Requirement | Priority |
|----|-------------|----------|
| FRK-AUTH-010 | `BasicAuth(realm string, verify func(identifier, password string) (*Identity, error))` parses RFC-7617 credentials, invokes the verifier, stores the identity (Method `"basic"`); failures answer `401` with `WWW-Authenticate: Basic realm="<realm>"` envelope. Malformed headers also 401 without invoking the verifier. | Must |

### JWT

| ID | Requirement | Priority |
|----|-------------|----------|
| FRK-AUTH-020 | `SignJWT(secret []byte, claims Claims) (string, error)` produces compact HS256 JWS (base64url, no padding); `ParseJWT(secret, token) (Claims, error)` pins `alg=HS256` (rejects others/`none`), verifies signature constant-time, enforces `exp` when present, returns parsed claims. | Must |
| FRK-AUTH-021 | `JWTAuth(secret, opts...)` extracts `Authorization: Bearer <jwt>` (or configured cookie), verifies, stores identity (Method `"jwt"`, Subject from `sub`, Roles merged from `roles` array / `role` scalar). Options: `ContinueOnMissing` (anonymous passthrough for optional auth), `RequiredClaim(key, value)` equality checks. Missing/invalid tokens ⇒ 401 unless ContinueOnMissing matched only-missing case. | Must |

### Policy gates

| ID | Requirement | Priority |
|----|-------------|----------|
| FRK-AUTH-030 | `Policy func(*Request) error`. `Require(policies ...Policy) Middleware` runs them in order before the handler; first error maps through `Error()` (e.g. 401/403 envelopes). | Must |
| FRK-AUTH-031 | Built-ins: `Authenticated()` (identity present else 401), `WithRoles(roles ...string)` (identity carries at least one, else 403). | Must |
| FRK-AUTH-032 | `AfterRequest(fn func(r *Request, status int))` observes the completed response status (audit log hook). | Must |
| FRK-AUTH-033 | `PolicySet map[string]Policy` with `.Require(names ...string) Middleware` resolving by name; unknown name panics at registration-time construction (developer error). | Should |

## 5. Non-Functional

- NFR1 — stdlib-only; timing-safe comparisons everywhere secrets are compared.
- NFR2 — Middlewares compose unchanged with `Use`/`Group(prefix, mws...)`/`RouteBuilder.Use`.
- NFR3 — Table-tested: credential matrix, JWT tamper/expiry/alg-swap matrix, gate order.

## 6. Success Criteria

- S1 — Basic flow: valid creds ⇒ handler sees identity; bad password ⇒ 401 + challenge header; garbage header ⇒ 401 (verifier untouched).
- S2 — JWT flow: round-trip claims incl. roles; tampered payload/signature ⇒ 401; `exp` past ⇒ 401; header claiming `alg:none` ⇒ 401.
- S3 — Gates: `Group("/admin", JWTAuth(sec), Require(Authenticated(), WithRoles("admin")))` admits admin JWT, rejects anonymous 401 and non-admin 403; `AfterRequest` records final status.
- S4 — Gates green: gofmt/vet/test in framework.

## 7. Related Specs

| SpecID | Title |
| ------ | ----- |
| KWF-WEB-P3V8X | Expressive HTTP (composition target) |
| KWF-WEB-R9T4C | Security & state stack (error envelopes, middleware patterns) |
