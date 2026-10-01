# Specification — Automatic Asset Injection

| Field       | Value                                       |
| ----------- | ------------------------------------------- |
| SpecID      | KWF-AI7Q2                              |
| Title       | Automatic CSS/JS Asset Injection         |
| Status      | Draft                                       |
| Date        | 2026-10-01                                  |
| Author      | Krewire Contributors                         |
| Domain      | Frameworks — Web — SSG — Assets             |

## 1. Context

The SSG already ships an asset pipeline (KWF-DR5YU) that copies, minifies, and
fingerprints assets, and an `asset("path")` helper that resolves a logical name
to its fingerprinted URL. What every layout still had to do was write the
`<link>`/`<script>` tags by hand, or call `{{assetLinks .AssetVersion}}` and
accept a fixed list of well-known file names.

## 2. Problem Statement

**Developers building a Krewire site face these pain points:**

- **Manual imports in every layout** — adding `public/assets/site.js` requires
  editing each layout by hand, or remembering the `assetLinks` helper exists
- **Stale hardcoded lists** — `assetLinks` enumerates a fixed set of names
  (`style.css`, `tailwind.css`, `forge.css`, `theme.css`, `app.js`, `docs.js`);
  any new asset is silently omitted until the framework is patched
- **Silent 404s** — a hand-written tag survives after the asset is renamed or
  removed; nothing reports it
- **No way to opt out** — a project that manages its own tags has no supported
  escape hatch

## 3. Goals

- G1 — Zero-config: a rendered document references every site CSS and JS asset
  without the layout naming any of them
- G2 — No duplicates: a tag already present in the template is never repeated
- G3 — Correct placement: stylesheets in `<head>`, scripts in `<head>` by
  default (first-paint theme scripts) or end of `<body>` on request
- G4 — Manifest-aware: fingerprinted pipeline output is linked through its
  manifest URL, never its source name
- G5 — Escape hatches: `auto_assets.enabled: false` and an `exclude` list

## 4. Non-Goals

- NG1 — Not a bundler: no module graph, concatenation, or dependency tracking
- NG2 — Does not inline assets into the HTML
- NG3 — Images, fonts, and favicons stay template-driven (`asset()` / `{{asset}}`);
  only `.css` and `.js` are injected

## 5. Requirements

### 5.1 Injection

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-AS-050  | Injection is enabled by default; layouts need no asset tags         | Must     |
| FRK-AS-051  | Every registered `assets/*.css` is appended to `<head>` as `<link rel="stylesheet">` | Must |
| FRK-AS-052  | Every registered `assets/*.js` is appended as `<script src>` to `<head>` (default) or end of `<body>` when `js_placement: body` | Must |
| FRK-AS-053  | Injection runs on the shared render path, so `Build`, `BuildIncremental`, `Handler`, and `RenderPage` emit identical documents | Must |
| FRK-AS-054  | A `<link>`/`<script>` URL already in the document is not injected again, matched with or without the `?v=` query | Must |
| FRK-AS-055  | URLs come from the pipeline manifest, so fingerprinted output is linked by its hashed name | Must |

### 5.2 Selection

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-AS-060  | Only asset names under `assets/` are injected — the prefix the emitted URL and the written output path agree on | Must |
| FRK-AS-061  | The generated scoped stylesheet `assets/style.css` is injected when any component or layout carries styles | Must |
| FRK-AS-062  | Assets registered via `ScriptAsset` (a `<script>` block inside one `.kiw` layout or page) are never injected site-wide | Must |
| FRK-AS-063  | `DeclareAsset` marks externally produced assets (plugin output such as Tailwind) for injection without the SSG writing them | Must |

### 5.3 Configuration

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-AS-070  | `auto_assets.enabled: false` in `krewire.yaml` disables injection entirely | Must |
| FRK-AS-071  | `auto_assets.exclude` accepts a full asset name, a base name, or a glob (`*.min.css`) | Must |
| FRK-AS-072  | `auto_assets.js_placement` selects `head` (default) or `body` for injected scripts | Must |
| FRK-AS-073  | `{{assetLinks .AssetVersion}}` remains available for templates that want an explicit, inline tag list | Should |
| FRK-AS-074  | `auto_assets.order` pins one asset to a cascade `layer` (`scoped`, `vendor`, `component`, `theme`, `book`, `page`), matched by full asset path or base name; an unknown layer name is reported through `AutoAssetErrors` without failing the build | Must |

### 5.4 Ordering

| ID          | Requirement                                                       | Priority |
| ----------- | ----------------------------------------------------------------- | -------- |
| FRK-AS-080  | Assets are injected in cascade-layer order, never alphabetical: `scoped` → `vendor` → `component` → `theme` → `book` → `page` | Must |
| FRK-AS-081  | The order is a total order — layer, then explicit order, then registration sequence, then asset name — so no two builds differ | Must |
| FRK-AS-082  | One table (`builtinAssets`) is the single source of the default order, read by automatic injection, `{{assetLinks}}`, and the exported plan, so the three cannot drift | Must |
| FRK-AS-083  | A builtin asset is linked only when the build actually produces it (registered, declared by a plugin, or generated), so no page ships a dead `<link>` | Must |
| FRK-AS-084  | `InjectedAssets` returns the resolved CSS/JS URLs, so a host rendering a book into the same output reuses the plan instead of copying the asset list | Must |

## 6. Non-Functional Requirements

- NFR1 — Deterministic: the order is a total order resolved from one table, not
  derived from map iteration
- NFR2 — Pure Go; the document is rewritten with `golang.org/x/net/html`
- NFR3 — Backward compatible: an existing layout that already links its assets
  renders with the same tags, not doubled

## 7. Success Criteria

- S1 — A layout containing only `<html><head><title>…</title></head><body>{{.Content}}</body></html>` renders a fully styled, interactive site
- S2 — Adding `public/assets/site.js` makes it appear in every page with no template edit
- S3 — `kiw build` on a Tailwind project links `/assets/tailwind.css` without a manual `<link>`
- S4 — A layout that already writes `<link rel="stylesheet" href="/assets/style.css">` produces that tag exactly once

## 8. Related Specifications

| SpecID      | Relationship |
| ----------- | ------------ |
| [KWF-DR5YU](./KWF-SSG-DR5YU-ssg-asset-pipeline.md) | Asset transforms and manifest this feature links |
| [KWF-DF3PL](./KWF-SSG-DF3PL-file-site-pipeline.md) | File-based sites loaded from `krewire.yaml` |
| [KWF-PT8OD](./KWF-SSG-PT8OD-static-site-generator.md) | Core SSG |