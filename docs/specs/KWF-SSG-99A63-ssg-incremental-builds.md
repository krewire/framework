# Specification — SSG Incremental Builds & Dependency Graph

| Field       | Value                                       |
| ----------- | ------------------------------------------- |
| SpecID      | KWF-99A63                              |
| Title       | SSG Incremental Builds & Dependency Graph   |
| Status      | Draft                                       |
| Date        | 2026-08-19                                  |
| Author      | Krewire Contributors                         |
| Domain      | Frameworks — Web — SSG — Build              |

## 1. Context

The current SSG rebuilds the entire site on every change. For sites with
hundreds of pages, this adds seconds of latency to the dev loop and CI builds.

## 2. Problem Statement

**Developers working on medium-to-large Krewire sites face these pain points:**

- **Full rebuild on every save** — changing one Markdown file or component
  template triggers a complete site rebuild; dev loop slows to 2–5s as the
  site grows
- **No change detection** — the build has no concept of what files a page
  depends on (components, layouts, data files, assets); it cannot skip
  unaffected pages
- **CI builds waste time** — every deploy rebuilds everything even when only
  one blog post changed

**Result:** Teams avoid Krewire for larger projects because the feedback loop
doesn't scale.

## 3. Goals

- G1 — Build a file dependency graph at build time: which source files
  (components, layouts, Markdown, data, assets) each output page depends on
- G2 — Incremental `Build`: only re-render pages whose dependency signatures
  changed since last build
- G3 — Persist dependency graph + content hashes to disk (`build-cache/`)
  for incremental CI builds and dev server restarts
- G4 — Dev server uses incremental render: on file change, re-render only
  affected pages and push updates via HMR (separate spec)
- G5 — Zero-config: dependency tracking works automatically from template
  `component` calls, layout references, asset paths, and data access

## 4. Non-Goals

- NG1 — Not a full build system (no DAG execution, no parallel job scheduling
  beyond page-level)
- NG2 — Not distributed caching; local disk cache only for MVP
- NG3 — Not hot module replacement for CSS/JS (separate spec)

## 5. Requirements

### 5.1 Dependency Graph

| ID            | Requirement                                                          | Priority |
| ------------- | -------------------------------------------------------------------- | -------- |
| FRK-INC-001   | During `prepare()`, record every `component` call, layout reference, | Must     |
|               | asset path, and data access per page                                 |          |
| FRK-INC-002   | Compute content hash (SHA256) for every source file at build start  | Must     |
| FRK-INC-003   | Persist graph + hashes to `build-cache/depgraph.json` after build   | Must     |

### 5.2 Incremental Build

| ID            | Requirement                                                          | Priority |
| ------------- | -------------------------------------------------------------------- | -------- |
| FRK-INC-010   | `Site.BuildIncremental(outDir, cache)` re-renders only pages whose  | Must     |
|               | dependency hashes differ from cache                                  |          |
| FRK-INC-011   | New pages (not in cache) are rendered; deleted pages remove output  | Must     |
| FRK-INC-012   | Asset pipeline changes invalidate dependent pages                   | Must     |
| FRK-INC-013   | Config/theme changes invalidate all pages                           | Must     |

### 5.3 Cache Persistence

| ID            | Requirement                                                          | Priority |
| ------------- | -------------------------------------------------------------------- | -------- |
| FRK-INC-020   | Cache directory: `outDir/.krewire-cache/` (gitignored)               | Must     |
| FRK-INC-021   | Cache format: JSON with version header; corrupted cache = full rebuild | Must |
| FRK-INC-022   | `krewire build --force` ignores cache and writes fresh               | Must     |

### 5.4 Dev Server Integration

| ID            | Requirement                                                          | Priority |
| ------------- | -------------------------------------------------------------------- | -------- |
| FRK-INC-030   | `Site.Handler()` uses incremental render on file change (watcher)   | Must     |
| FRK-INC-031   | Cache shared between `krewire build` and `krewire dev`                | Must     |

## 6. Non-Functional Requirements

- NFR1 — Cache read/write overhead < 50ms for typical sites
- NFR2 — Deterministic: same inputs → same outputs (cache hit) / same changed pages (cache miss)
- NFR3 — Quality gates pass

## 7. Success Criteria

- S1 — 100-page site: first build 2.1s, incremental change to one post 0.15s
- S2 — CI cache hit on unchanged PR rebuilds in < 5s
- S3 — Dev server re-renders only the changed page on component edit

## 8. Related Specifications

| SpecID      | KWF-99A63                              |
| --------- | ----------------------------------------------- |
| [KWF-PT8OD](./KWF-SSG-PT8OD-static-site-generator.md) | Core SSG |
| [KWF-D57UK](./KWF-SSG-D57UK-ssg-markdown-content-pipeline.md) | Content Pipeline |
| [KWF-DR5YU](./KWF-SSG-DR5YU-ssg-asset-pipeline.md) | Asset Pipeline |
| [KWF-209JV](./KWF-SSG-209JV-ssg-live-reload-hmr.md) | Live Reload / HMR |