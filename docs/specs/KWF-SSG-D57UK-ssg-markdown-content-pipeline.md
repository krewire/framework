# Specification — SSG Markdown Content Pipeline & Collections

| Field       | Value                                       |
| ----------- | ------------------------------------------- |
| SpecID      | KWF-D57UK                              |
| Title       | SSG Markdown Content Pipeline & Collections |
| Status      | Draft                                       |
| Date        | 2026-08-19                                  |
| Author      | Krewire Contributors                         |
| Domain      | Frameworks — Web — SSG — Content            |

## 1. Context

The current SSG (`web/ssg`) renders components and layouts from templates but
has no content layer. Authors must write raw HTML in templates or config.

## 2. Problem Statement

**Developers building content-driven sites with Krewire face these pain points:**

- **No Markdown authoring** — must write raw HTML in Go templates for every blog post, documentation page, or changelog entry
- **No content collections** — cannot group related pages (blog, docs, changelog) with shared layout and validation; every page is a manual config entry
- **No frontmatter** — cannot attach metadata (date, tags, draft status, SEO description) to pages
- **No draft/future publishing** — everything in `content/` goes live immediately; no staging workflow
- **No template helpers** — cannot use `{{ markdown "..." }}` for inline Markdown rendering

**Result:** Teams choose Astro, Hugo, or Next.js instead, because Krewire doesn't solve the content authoring problem.

## 3. Goals

- G1 — Write content in Markdown with YAML frontmatter; get typed page data + rendered HTML automatically
- G2 — Define content collections (blog, docs, changelog) in `ssg.yaml` with shared layout, permalink pattern, and schema
- G3 — `{{ markdown "..." }}` helper in templates for inline Markdown rendering
- G4 — Draft posts stay local; future-dated posts publish on their date; dev server shows both
- G5 — Byte-identical output between `krewire build` (static) and `krewire dev` / `krewire run` (dynamic SSR)

## 4. Non-Goals

- NG1 — Not a headless CMS or remote data layer (separate spec)
- NG2 — Not replacing mdbind; mdbind handles book-style manuscripts, this handles site-style content collections
- NG3 — No incremental builds in this spec (separate spec)

## 5. Requirements

### 5.1 Markdown + Frontmatter

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-MD-001  | `.md` files with `---` YAML frontmatter parse to page data + trusted HTML (`template.HTML`) | Must |
| FRK-MD-002  | Built-in frontmatter fields: `title`, `date`, `draft`, `tags`, `description`, `slug`, `layout`; unknown fields pass through | Must |
| FRK-MD-003  | Flexible date parsing (RFC3339, ISO8601, common formats), UTC default | Must |

### 5.2 Content Collections

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-MD-010  | `collections:` in `ssg.yaml`: each has `name`, `dir`, `pattern`, `layout`, `output` pattern, optional `schema` | Must |
| FRK-MD-011  | Collection pages auto-generate from matching files; path from `slug` or filename | Must |
| FRK-MD-012  | `output` pattern supports `:slug`, `:year`, `:month`, `:day`, `:collection` placeholders | Must |
| FRK-MD-013  | Collection `schema` (optional) validates frontmatter against required/optional fields | Should |

### 5.3 Template Helpers

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-MD-020  | `{{ markdown "string" }}` → trusted HTML using same parser | Must |
| FRK-MD-021  | `{{ range .Site.Collections.blog }}` yields pages with `.Title`, `.Date`, `.Permalink`, `.Data`, `.Content` | Must |

### 5.4 Draft / Future Publishing

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-MD-030  | `draft: true` excludes from production build; `IncludeDrafts: true` in config enables for dev | Must |
| FRK-MD-031  | Future `date` excludes until that date; `IncludeFuture: true` enables for dev | Must |
| FRK-MD-032  | Dev server (`Site.Handler()`) respects both flags from config | Must |

### 5.5 SSR Parity

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-MD-040  | Markdown page built via SSG = byte-identical to dynamic SSR via `web.App` (KWF-0F2EB) | Must |
| FRK-MD-041  | `web.App` loads collection files on-demand for dynamic rendering | Must |

## 6. Non-Functional Requirements

- NFR1 — `github.com/gomarkdown/markdown` (no CGo)
- NFR2 — Frontmatter uses existing `gopkg.in/yaml.v3`
- NFR3 — Single-pass build for MVP; incremental is separate spec
- NFR4 — Quality gates: `gofmt`, `go vet ./...`, `go test ./...`

## 7. Success Criteria

- S1 — `ssg.yaml` with `blog` collection builds from `.md` files in `content/blog/` with correct permalinks
- S2 — Draft posts excluded from production, visible in dev
- S3 — `{{ markdown "## Hello" }}` renders `<h2>Hello</h2>`
- S4 — Layout iterates `.Site.Collections.blog` with correct URLs and frontmatter data

## 8. Related Specifications

| SpecID      | KWF-D57UK                              |
| --------- | ----------------------------------------------- |
| [KWF-PT8OD](./KWF-SSG-PT8OD-static-site-generator.md) | Core SSG (components, layouts, scoped CSS) |
| [KWF-0F2EB](./KWF-WEB-0F2EB-server-frontend-pipeline.md) | Shared page model (SSR = static parity) |
| [KWF-F2TQC](./KWF-JS-F2TQC-js-ts-framework-integration.md) | Hydration-ready output |