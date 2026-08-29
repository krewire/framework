# Specification — HTTP Security Headers, CSRF/XSS, Cache, Session & Cookies

| Field  | Value |
| ------ | ----- |
| SpecID | KWF-WEB-R9T4C |
| Title  | HTTP Security & State: Headers, CSRF, XSS, Cache, Session, Cookie |
| Status | Draft |
| Date   | 2026-08-23 |
| Author | Krewire Contributors |
| Domain | Framework — Web — HTTP Security & State |

## 1. Context

Web apps built on `framework/web` need production-grade browser-facing defenses
and state management. Today the router has recover/access-log middleware but no
security headers, no CSRF, no cache policies, and no session story; cookies are
written through raw `http.SetCookie`. XSS is primarily prevented by
`html/template` auto-escaping; a defense-in-depth layer (CSP, nosniff,
frame-options) is missing.

This spec adds five cohesive middlewares/helpers to `framework/web`, composing
with the expressive layer (`Group`, `Use`, `H`) and reusing `storage.KV` as a
session backend.

## 2. Problem Statement

- **Current pain:** No cohesive security/state middlewares — `SecurityHeaders`, `CORS`, `CSRF`, `Session`, `Cookie`/`JWT`, `Cache` are missing or scattered. Each `app` hard-codes `X-Frame-Options`/`SameSite` with different defaults and no `Secure`/`HttpOnly` audit.
- **Affected consumers:** `framework/web` authors, `app` teams serving SSR+API on cheap infra, and security reviewers.
- **Cost of leaving unsolved:** Security headers and cookie attributes diverge per service, `Session` is not `net/http`-typed, and `framework/web` cannot claim `secure by default`.

## 3. Goals

- G1 — One-line security headers with sane defaults and explicit configuration.
- G2 — CSRF double-submit tokens bound to the session when present, verified in constant time on unsafe methods, exposed to handlers/templates.
- G3 — Cache policies as composable middleware (`NoStore`, `MaxAge`, immutable assets).
- G4 — Server-side sessions: pluggable store (memory + any `storage.KV`), lazy TTL, fixation-safe regeneration, sliding expiration, HttpOnly cookie.
- G5 — Fluent cookie builder and `Request.CookieVal`.

## 4. Non-Goals

- NG1 — No HTML sanitizer (template escaping + CSP remain the XSS strategy; a sanitizer would need fuzz-hardened maintenance).
- NG2 — No encrypted/signed cookies (future); session payloads stay server-side.
- NG3 — No distributed session locking.

## 5. Requirements

### Security headers & XSS

| ID | Requirement | Priority |
|----|-------------|----------|
| FRK-SEC-001 | `SecurityHeaders(opts...)` sets defaults `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: strict-origin-when-cross-origin`, CSP `default-src 'self'`; options override CSP, enable HSTS (with max-age), relax frame mode, add Permissions-Policy. Idempotent (never duplicates). | Must |
| FRK-SEC-002 | `StripTags(s) string` strips HTML tags for plain-text fields (defense-in-depth beside escaping). | Should |

### CSRF

| ID | Requirement | Priority |
|----|-------------|----------|
| FRK-SEC-010 | `CSRF(opts...)` issues a random token cookie (`XSRF-TOKEN` default, `SameSite=Lax`) on safe requests when absent; verifies unsafe methods (POST/PUT/PATCH/DELETE) comparing header `X-CSRF-Token` or form field `csrf_token` against the cookie value — or the session-bound token when a session exists — via constant-time compare; failures return 403 envelope. Requests bearing `Authorization` are exempt. | Must |
| FRK-SEC-011 | Verified/current token reachable in handlers via `*Request.CSRFToken()` (context) and `CSRFFrom(context)`. | Must |

### Cache

| ID | Requirement | Priority |
|----|-------------|----------|
| FRK-SEC-020 | `NoStore()` middleware emits `Cache-Control: no-store, private`. `MaxAge(seconds, public bool)` emits `public/private, max-age=n` (+`s-maxage` when public>0 given separately). `Immutable(maxAge)` for fingerprinted assets. Raw escape hatch `CacheControl(value)`. | Must |

### Session

| ID | Requirement | Priority |
|----|-------------|----------|
| FRK-SEC-030 | `Session` holds `ID`, string-keyed `Data` (JSON-serializable values), methods `Get/Set/Delete/Rotate` (fixation defense) and dirty tracking. | Must |
| FRK-SEC-031 | `SessionStore` interface (`Load/Save/Delete`); `NewMemorySessionStore()` provided; `KVSessionStore(kv storage.KV)` adapts any KV backend under namespace `sessions/`. Expired entries treated as absent (lazy TTL). | Must |
| FRK-SEC-032 | `Sessions(store, opts...)` middleware resolves the cookie (`kiw_session` default, HttpOnly, SameSite=Lax, Path=/), exposes `*Request.Session()`, persists dirty sessions after the handler with sliding TTL (default 24h), sets/clears cookie accordingly. | Must |
| FRK-SEC-033 | `Rotate()` regenerates the ID, moves data, deletes the old record, and rewrites the cookie. | Must |

### Cookies

| ID | Requirement | Priority |
|----|-------------|----------|
| FRK-SEC-040 | Fluent builder `Cookie(name, value)` chaining `Path/Domain/MaxAge/Secure/HttpOnly/SameSite`, finished with `Write(w)`; `DeleteCookie(w, name, path...)`. `Request.CookieVal(name) string`. | Must |

## 5. Non-Functional

- NFR1 — stdlib + `golang.org/x/crypto` avoided: tokens via `crypto/rand`; compare via `crypto/subtle`.
- NFR2 — Middlewares compose with `Use`/`Group`/`H` unchanged; all behavior table-tested including full `SecurityHeaders→Sessions→CSRF` stack.

## 7. Success Criteria

- S1 — Stack test: POST without token ⇒ 403; with issued token in header ⇒ 200; session-bound token survives Rotate invalidating the old one; expired session behaves as new.
- S2 — KV-backed session equals memory-backed behavior in the shared contract test.
- S3 — Gates green: `gofmt`, `go vet`, `go test ./...` in framework.

## 7. Related Specs

| SpecID | Title |
| ------ | ----- |
| KWF-WEB-P3V8X | Expressive HTTP layer this composes with |
| KWF-AST-K7Q2M | App storage (`storage.KV` reused as session backend) |
