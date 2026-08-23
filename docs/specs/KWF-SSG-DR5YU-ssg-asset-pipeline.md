# Specification — SSG Asset Pipeline

| Field       | Value                                       |
| ----------- | ------------------------------------------- |
| SpecID      | KWF-DR5YU                              |
| Title       | SSG Asset Pipeline (Images, Hashing, Minification) |
| Status      | Draft                                       |
| Date        | 2026-08-19                                  |
| Author      | Krewire Contributors                         |
| Domain      | Frameworks — Web — SSG — Assets             |

## 1. Context

The current SSG passes assets through verbatim (`Assets map[string]string`).

## 2. Problem Statement

**Developers deploying Krewire sites face these pain points:**

- **Unoptimized images** — large PNG/JPEG files shipped as-is; no automatic WebP/AVIF conversion or responsive resizing
- **No cache busting** — `style.css` and `app.js` cached aggressively by browsers; deploying updates requires manual filename changes or waiting for cache expiry
- **Bloated CSS/JS** — no minification; production bundles ship with whitespace and comments
- **Manual asset references** — templates hardcode `/assets/style.css`; changing to fingerprinted names requires find-and-replace across templates
- **No asset manifest** — cannot programmatically verify what assets were emitted

## 3. Goals

- G1 — Asset pipeline with transforms configured in `ssg.yaml`
- G2 — Image optimization: auto WebP/AVIF, responsive resizing, quality control
- G3 — CSS/JS minification (opt-in)
- G4 — Content-hash fingerprinting: `style.css` → `style.a1b2c3d4.css`
- G5 — Asset manifest (`manifest.json`) + `asset("path")` template helper for automatic fingerprinted URLs
- G6 — Dev server serves fingerprinted assets without full rebuild

## 4. Non-Goals

- NG1 — Not a bundler (no module graph, no code splitting, no tree-shaking)
- NG2 — Not a CDN or runtime image service; transforms are build-time only
- NG3 — No source maps in MVP

## 5. Requirements

### 5.1 Pipeline Configuration

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-AS-001  | `ssg.yaml` adds `assets.pipeline:` with transforms by glob pattern | Must |
| FRK-AS-002  | Built-in transformers: `resize`, `webp`, `avif`, `minify-css`, `minify-js`, `hash`, `copy` | Must |
| FRK-AS-003  | Pipeline runs at build; sources in `assets/` or collection dirs; output to `outDir/assets/` | Must |

### 5.2 Image Optimization

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-AS-010  | `resize` (width/height, fit mode), `format` (webp/avif/jpeg/png), `quality` (1-100) | Must |
| FRK-AS-011  | Multiple outputs per source: `hero.jpg` → `hero.webp`, `hero.avif`; original preserved unless `original: false` | Should |
| FRK-AS-012  | Pure Go: `github.com/disintegration/imaging` + native encoders | Must |

### 5.3 Minification

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-AS-020  | CSS/JS minification via `github.com/tdewolff/minify/v2` | Must |
| FRK-AS-021  | Hash computed from minified output (not source) | Must |

### 5.4 Fingerprinting & Manifest

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-AS-030  | SHA256 (first 8 hex) appended before extension: `style.css` → `style.a1b2c3d4.css` | Must |
| FRK-AS-031  | `manifest.json` at output root: `{"style.css": "/assets/style.a1b2c3d4.css"}` | Must |
| FRK-AS-032  | `asset("path")` template helper returns fingerprinted URL from manifest | Must |
| FRK-AS-033  | CSS `url()` references rewritten to fingerprinted paths | Should |

### 5.5 SSR Parity

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-AS-040  | Dev server serves fingerprinted assets from manifest (built on first request) | Must |
| FRK-AS-041  | `web.App` asset serving uses same manifest | Must |

## 6. Non-Functional Requirements

- NFR1 — Pure Go image processing (no CGO) for cross-compilation
- NFR2 — Deterministic: same input → same hash
- NFR3 — Parallelizable per asset
- NFR4 — Quality gates pass

## 7. Success Criteria

- S1 — `ssg.yaml` with `assets.pipeline` produces fingerprinted assets + manifest
- S2 — `asset("style.css")` in template resolves to fingerprinted URL
- S3 — Images in `assets/images/` auto-converted to WebP + resized variants
- S4 — CSS minified and fingerprinted; `url()` refs rewritten

## 8. Related Specifications

| SpecID      | KWF-DR5YU                              |
| --------- | ----------------------------------------------- |
| [KWF-PT8OD](./KWF-SSG-PT8OD-static-site-generator.md) | Core SSG |
| [KWF-D57UK](./KWF-SSG-D57UK-ssg-markdown-content-pipeline.md) | Markdown Content Pipeline |
| [KWF-0F2EB](./KWF-WEB-0F2EB-server-frontend-pipeline.md) | Shared page model |