# Specification — File-Based Site Pipeline ( `.kiw` Modules)

| Field       | Value                                   |
| ----------- | --------------------------------------- |
| SpecID      | KWF-DF3PL                               |
| Title       | File-Based Site Pipeline (`.kiw` Modules) |
| Status      | Draft                                   |
| Date        | 2026-08-20                              |
| Author      | Krewire Contributors                     |
| Domain      | Frameworks — Web — SSG — Authoring      |

## 1. Context

The `web/ssg` package builds sites from a declarative `krewire.yaml` whose
`ssg:` key inlines component markup, layout markup, styles, pages, and theme.
Content authored this way does not scale: a site with one hundred pages must
embed one hundred markup blocks inside a single YAML document.

This specification refounds authoring on source **files** in the Astro model:

- `pages/` — one `.kiw` file per route, split by frontmatter + template + scoped
  style. Routes are derived from the filesystem (file-based, extensionless).
- `components/` — reusable `.kiw` modules, invoked from templates by name.
- `layouts/` — `.kiw` page shells with a content slot.
- `content/` — Markdown collections (frontmatter + body) auto-discovered from
  directories; `pages/*/[slug].kiw` consume them.
- `public/` — files copied verbatim into the output.

`krewire.yaml` is reduced to **site metadata** (`name`, `kind`, `title`,
`description`, `theme`, `output`) and is written exactly once by `krewire new`
or `krewire init`. After scaffolding no one edits it by hand again.

## 2. Problem Statement

Markup embedded in `krewire.yaml` (first generation: `ssg:` key) is a
conceptual dead-end:

- Large sites are unmaintainable (one YAML per hundred pages).
- IDE tooling, git diffs, and code review degrade when HTML/CSS lives in a
  config block scalar.
- The pipeline is driven by config instead of by content, so the framework
  cannot reason about files, routes, or partial rebuilds.
- mdbind's `manuscript/` files and `ssg:` config describe two different
  authoring worlds instead of one file-based model.

## 3. Goals

- G1 — Author pages, components, and layouts as **files**, never as YAML
  inline markup.
- G2 — Derive routes from the `pages/` filesystem tree with file-based,
  extensionless URLs emitted as sibling `.html` files.
- G3 — Make `krewire.yaml` metadata-only and write-once.
- G4 — Unify site authoring behind **one file-based engine**; markdown book
  content (`manuscript/`) becomes a leaf of the same pipeline.
- G5 — Remain Go-first (Livewire ethos): templates are `html/template`, styles
  are scoped CSS, and all interactivity is framework-generated — the author
  never leaves Go and never hand-writes JavaScript.

## 4. Non-Goals

- NG1 — Client-side islands/components in this phase; hydratable-ready output is
  preserved (see KWF-F2TQC).
- NG2 — A content-management system, admin, or cloud sync.
- NG3 — Replacing Go-native `ui.Registry` components; file modules and Go
  components coexist, files resolving by name first.
- NG4 — Graceful migration paths for legacy `ssg:` inline configs; the inline
  shape is removed (see KWF-PT8OD FRK-SSG-010/016/018, superseded).
- NG5 — Rejecting Astro's ecosystem anti-patterns without copying its
  machinery: no content-loader/config-DSL API (folder-is-collection is the
  only collection model), no ever-growing central config (metadata-only
  `krewire.yaml` is a hard ceiling), no BYOF adapter matrix, no TypeScript
  toolchain, no dual asset trees (one `public/`), one markdown renderer, and a
  deterministic frontmatter rule (file starts with `---\n` → frontmatter).

## 5. Requirements

### Source Layout

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-FLS-001 | A site is the union of `pages/`, `components/`, `layouts/`, `content/`, and `public/` directories in the project root; absence of a directory means it contributes nothing. | Must |
| FRK-FLS-002 | `krewire.yaml` declares only `project.name`, `project.kind`, `title`, `description`, `theme`, and `output`. No `ssg:` section exists. | Must |
| FRK-FLS-003 | The engine loads a site from the project root via a single entry point (e.g. `ssg.LoadFromDir(root)` → `*Site`); the former `Config`/`BuildFromConfig` becomes an internal detail or is deleted. | Must |

### `.kiw` Module Format

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-FLS-010 | A `.kiw` file is an optional `---frontmatter---` YAML block, a `html/template` body, and zero or more top-level `<style>` blocks extracted as that module's scoped CSS. | Must |
| FRK-FLS-011 | Frontmatter maps to template data for pages/components/layouts: `title`, `description`, `layout`, `draft`, `date`, `tags`, `slug`, plus arbitrary extra fields. | Must |
| FRK-FLS-012 | Components and layouts register under their base filename (e.g. `components/Header.kiw` → `Header`) and are invoked from templates with the existing `component` helper. | Must |
| FRK-FLS-013 | Scoped CSS semantics are unchanged: rules rewrite under `[data-kiw-component="name"]` / `[data-kiw-layout="name"]`, with `:root` rules left global. | Must |
| FRK-FLS-014 | Files and directories whose names start with `_` are never routed as pages (partials convention) but remain available as components/layouts by name. | Must |

### Routing & Pages

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-FLS-020 | `pages/index.kiw` → `/`; `pages/about.kiw` → `/about`, emitted as `about.html`; `pages/docs/quickstart.kiw` → `/docs/quickstart`, emitted as `docs/quickstart.html`; `pages/docs/index.kiw` → `/docs`. No trailing slashes anywhere. | Must |
| FRK-FLS-021 | A page's `[slug].kiw` in `pages/<name>/` becomes a dynamic route materialized for every item in the `content/<name>` collection, at `/name/:slug`, emitted as `name/:slug.html`. | Must |
| FRK-FLS-022 | Page data merges site metadata (`Site.Title`, `Theme`), global `Data`, frontmatter, and — for `[slug]` pages — the matched collection item's params. | Must |
| FRK-FLS-023 | A page's `layout` frontmatter names a `layouts/` file; absence resolves to `layouts/Base.kiw` when present, else the built-in document shell. | Must |

### Content Collections

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-FLS-030 | Every directory under `content/` is a collection named after it; Markdown files with `---` frontmatter are items. Supersedes config-declared `collections:`. | Must |
| FRK-FLS-031 | Collection items expose `title`, `date`, `draft`, `tags`, `description`, `slug`, extra params, and rendered `Content`. Draft/future filtering is preserved. | Must |
| FRK-FLS-032 | Items without an explicit `layout` use `layouts/<collection>.kiw`, else `layouts/Post.kiw`, else `layouts/Base.kiw`. | Must |
| FRK-FLS-033 | A collection name directory may also hold `index.kiw` hosting listing pages over the collection data. | Should |

### Public & Assets

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-FLS-040 | `public/` files are copied verbatim to the output root, preserving paths; a file in `public/` overrides a machine-generated asset with the same path. | Must |
| FRK-FLS-041 | Framework-generated assets (`theme.css`, `ui.css`, collected `style.css`) are emitted to `assets/` as today; authors hook scripts via the existing script injection point (FRK-SSG-024). | Must |

### Go-First Behaviour

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-FLS-050 | The author writes no JavaScript: all interactivity (theme switching and any future framework scripts) is injected by the engine; template functions (`component`, `html`, `markdown`) cover data shaping. | Must |
| FRK-FLS-051 | The build and dev server render through the same `Site` render path, keeping byte-identical output (KWF-PT8OD FRK-SSG-019/020 preserved). | Must |

### Book-Mode Leaf

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-FLS-060 | A `manuscript/` directory (mdbind book) is loadable through the same file pipeline: chapters become a collection with a book layout and file-based navigation; no book-specific engine survives separately except the manuscript parser. | Must |

## 6. Non-Functional Requirements

- NFR1 — No `unsafe`; minimal deps (`golang.org/x/net/html` for HTML work; existing YAML + gomarkdown).
- NFR2 — Quality gates: `gofmt -l .`, `go vet ./...`, `go test ./...` pass in `framework`, `krewire`, `mdbind`.
- NFR3 — Deterministic ordering: collections sort by date descending, routes and CSS by name.

## 7. Success Criteria

- S1 — A site with `pages/index.kiw`, `components/Header.kiw`, `layouts/Base.kiw`, `public/` builds while `krewire.yaml` holds metadata alone.
- S2 — `pages/blog/[slug].kiw` + `content/blog/*.md` produce `/blog/<slug>` pages with scoped styles and correct per-item data.
- S3 — No markup exists anywhere in `krewire.yaml`.
- S4 — `layouts/Base.kiw`'s style is scoped; `component "Header"` output carries `data-kiw-component`.
- S5 — `gofmt`, `go vet`, `go test` green across `framework`, `krewire`, `mdbind`.
- S6 — The landing page (`krewire.github.io`) is rebuilt from the file-based layout with a metadata-only `krewire.yaml`.

## 8. Related Specifications

| SpecID    | Title                                                        |
| --------- | ------------------------------------------------------------ |
| [KWF-PT8OD](./KWF-SSG-PT8OD-static-site-generator.md) | Static Site Generator (engine internals; inline-config rows superseded) |
| [KWF-D57UK](./KWF-SSG-D57UK-ssg-markdown-content-pipeline.md) | SSG Markdown Content Pipeline & Collections (becomes FRK-FLS-030..033) |
| [KWF-M07QS](./KWF-WEB-M07QS-krewire-web-framework.md) | Krewire Web Framework   |
| [KWF-PPUWX](./KWF-UI-PPUWX-layout-ui-system.md)       | Layout & UI System      |
| [KWF-F2TQC](./KWF-JS-F2TQC-js-ts-framework-integration.md) | JS/TS Framework Integration |
| [KWM-FX9H2](https://github.com/krewire/mdbind/blob/main/docs/specs/KWM-BUILDER-FX9H2-mdbind-site-builder.md) | mdbind Site Builder (converges into FRK-FLS-060) |