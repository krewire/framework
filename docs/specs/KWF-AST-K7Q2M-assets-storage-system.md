# Specification — Assets, Resources & App Storage

| Field  | Value |
| ------ | ----- |
| SpecID | KWF-AST-K7Q2M |
| Title  | Static Assets, Resources & App Storage (`framework/assets`, `framework/storage`) |
| Status | Draft |
| Date   | 2026-08-23 |
| Author | Krewire Contributors |
| Domain | Framework — Web — Assets / Runtime — Storage |

## 1. Context

Krewire apps and sites need three storage-shaped concerns today, handled ad hoc:

- **Static assets** (CSS/JS/images) are served by raw `http.FileServer` mounts (`web.Router.Static`/`StaticFS`) with no caching headers, no fingerprinting, and no manifest for templates.
- **Resources** (embedded data files: configs, seeds, i18n strings) are read ad hoc via `embed.FS` at call sites.
- **App storage** (runtime key-value state: uploads, cache, session blobs) has no abstraction; every app invents its own.

This specification introduces two flat packages following the ecosystem's "one package = one concern" rule.

## 2. Goals

- G1 — `framework/assets`: one `Store` that unifies asset sources (directory, `embed.FS`) into a single namespace, serves them over HTTP with correct content types, `ETag`, and `Cache-Control`, supports content-hash fingerprinting with a template-facing manifest.
- G2 — `framework/storage`: a small context-aware KV contract with memory and filesystem backends, zero-cost when unused, idiomatic `(value, ok, error)` reads.
- G3 — Both integrate with the existing DI container (`app.Provider`) and HTTP layer without new dependencies beyond stdlib (+`gopkg.in/yaml.v3` already in use).
- G4 — Resources are just read-only sources: an `embed.FS` mounted in `assets.Store` is the resource mechanism; structured helpers (`assets.JSON`/`assets.YAML`) decode embedded documents.

## 3. Non-Goals

- NG1 — No CDN upload/sync, no image processing, no S3 backend (future).
- NG2 — No database-backed storage; KV covers MVP app state. Sessions/uploads build on it later.
- NG3 — No changes to `web/ssg` output pipeline (its `public/` remains authoritative for static export).

## 4. Requirements

### assets

| ID | Requirement | Priority |
|----|-------------|----------|
| FRK-AST-001 | `assets.NewStore()` returns an empty `*Store`; sources mount via `Mount(fsys fs.FS)` (first match wins). `MountDir(dir)` wraps `os.DirFS`. | Must |
| FRK-AST-002 | `Store.Open(name)` returns `(content []byte, contentType string, etag string, err error)`; content type from extension sniffing (`.css/.js/.svg/.png/.json/...` fallback `application/octet-stream`); ETag is strong SHA-256 of content. | Must |
| FRK-AST-003 | `Store.Handler()` serves GET/HEAD under any mount point: sets `ETag`, honors `If-None-Match` with `304`, sets `Cache-Control: public, max-age=0, must-revalidate` by default; `Store.HandlerImmutable()` emits `Cache-Control: public, max-age=31536000, immutable` for fingerprinted paths. | Must |
| FRK-AST-004 | `Store.Fingerprint(name)` returns `<base>.<hash8>.<ext>`; `Store.Manifest()` returns `name → fingerprinted path` for every known file; unknown names error. Manifest is JSON-marshalable for templates. | Should |
| FRK-AST-005 | `assets.JSON[T]`/`assets.YAML[T]` decode an embedded resource document into a typed value (resources concern). | Should |

### storage

| ID | Requirement | Priority |
|----|-------------|----------|
| FRK-AST-010 | `storage.KV` interface: `Get(ctx, key) ([]byte, bool, error)`, `Put(ctx, key, val []byte) error`, `Delete(ctx, key) error`, `List(ctx, prefix) ([]string, error)` sorted. Keys are `/`-separated; empty values allowed. | Must |
| FRK-AST-011 | `storage.NewMemory() *MemoryKV` — goroutine-safe map backend, zero allocations on miss. | Must |
| FRK-AST-012 | `storage.NewFile(root string) (*FileKV, error)` — filesystem backend mapping keys to paths under root (path-traversal safe); atomic writes via temp file + rename. | Must |
| FRK-AST-013 | `storage.Provider(kv KV) app.Provider` registers `KV` into the `app` container so modules resolve it via DI. | Should |

## 5. Non-Functional

- NFR1 — stdlib-only (plus existing yaml dep); `gofmt -l .`, `go vet ./...`, `go test ./...` green.
- NFR2 — Deterministic: `List` sorted; `Manifest` stable ordering when marshaled via sorted keys.

## 6. Success Criteria

- S1 — A store mounting `testdata/...` serves CSS with `text/css`, matching ETag → `304`, mismatched → `200`.
- S2 — Fingerprinted request hits `HandlerImmutable` with `immutable` cache header; manifest maps original name.
- S3 — Memory and File backends pass the same behavioral test suite (table-driven across backends).
- S4 — `storage.Provider` binds into `app.NewApp(storage.Provider(kv))` and resolves.

## 7. Related Specs

| SpecID | Title |
| ------ | ----- |
| KWF-M07QS | Krewire Web Framework (static mounts supersede raw FileServer usage over time) |
| KWF-C4087 | App assembly & DI container |
