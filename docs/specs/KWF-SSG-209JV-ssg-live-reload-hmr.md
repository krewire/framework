# Specification — SSG Live Reload & HMR

| Field       | Value                                       |
| ----------- | ------------------------------------------- |
| SpecID      | KWF-209JV                              |
| Title       | SSG Live Reload & Hot Module Replacement    |
| Status      | Draft                                       |
| Date        | 2026-08-19                                  |
| Author      | Krewire Contributors                         |
| Domain      | Frameworks — Web — SSG — Dev Experience     |

## 1. Context

The current `Site.Handler()` re-renders pages on-demand but the browser
must be manually refreshed. Modern SSGs (Astro, Vite, Hugo) push updates
instantly via WebSocket.

## 2. Problem Statement

**Developers using `krewire dev` face these pain points:**

- **Manual refresh required** — after saving a component, layout, or
  Markdown file, must switch to browser and press F5 to see changes
- **Lost scroll position / form state** — full page reload discards
  ephemeral UI state (scroll, open menus, form inputs)
- **No CSS hot update** — style changes require full reload; no
  stylesheet hot-swap

**Result:** Dev loop feels slow compared to Vite/Astro; mental context
switch between editor and browser.

## 3. Goals

- G1 — WebSocket-based live reload: on file change, dev server pushes
  "reload" message; client refreshes automatically
- G2 — CSS hot-swap: on `.css`/component style change, inject new
  stylesheet without full reload (preserves scroll, form state)
- G3 — Optional HMR for Markdown/content: replace page content fragment
  via `data-props` without layout re-render
- G4 — Zero-config: works out of the box with `krewire dev`
- G5 — Falls back gracefully: if WS fails, client falls back to polling
  or manual refresh

## 4. Non-Goals

- NG1 — Not full React/Vue/Svelte HMR (component state preservation);
  page-level replacement only
- NG2 — Not a custom client runtime; uses vanilla WS + DOM APIs
- NG3 — Not replacing incremental build (KWF-99A63); consumes its output

## 5. Requirements

### 5.1 WebSocket Server

| ID            | Requirement                                                          | Priority |
| ------------- | -------------------------------------------------------------------- | -------- |
| FRK-HMR-001   | Dev server (`Site.Handler()`) upgrades `/__krewire/hmr` to WebSocket | Must     |
| FRK-HMR-002   | On file change (via incremental build), broadcast `{"type":"reload"}` | Must   |
| FRK-HMR-003   | Broadcast `{"type":"style-update", "href":"/assets/style.a1b2.css"}` | Must     |
|               | for CSS changes                                                      |          |
| FRK-HMR-004   | Heartbeat ping every 15s; client reconnects on disconnect            | Must     |

### 5.2 Client Runtime (Injected)

| ID            | Requirement                                                          | Priority |
| ------------- | -------------------------------------------------------------------- | -------- |
| FRK-HMR-010   | Dev server injects `<script src="/__krewire/hmr.js"></script>` into   | Must     |
|               | every page (dev only)                                                |          |
| FRK-HMR-011   | Client connects to WS, on `reload` → `location.reload()`             | Must     |
| FRK-HMR-012   | On `style-update`: fetch new stylesheet, replace `<link>` href,      | Must     |
|               | no page reload                                                       |          |
| FRK-HMR-013   | On `content-update`: replace page root innerHTML via `data-props`    | Should   |
|               | diff (future)                                                        |          |

### 5.2 CSS Hot-Swap

| ID            | Requirement                                                          | Priority |
| ------------- | -------------------------------------------------------------------- | -------- |
| FRK-HMR-020   | Fingerprinted stylesheet URL changes on content hash change          | Must     |
| FRK-HMR-021   | Client fetches new CSS, creates new `<link>`, swaps old one          | Must     |
| FRK-HMR-022   | Preserves scroll, form inputs, focus, CSS animations                 | Must     |

### 5.3 SSR Parity

| ID            | Requirement                                                          | Priority |
| ------------- | -------------------------------------------------------------------- | -------- |
| FRK-HMR-030   | Production build (`krewire build`) does NOT inject HMR script/WS      | Must     |
| FRK-HMR-031   | Dev-only flag controls injection (`Site.HMR(true)`)                 | Must     |

## 6. Non-Functional Requirements

- NFR1 — WS connection < 50ms latency on localhost
- NFR2 — Client runtime < 2KB gzipped
- NFR3 — No external deps; vanilla JS + WebSocket API
- NFR4 — Quality gates pass

## 7. Success Criteria

- S1 — Edit component style → browser updates CSS in < 200ms, no reload
- S2 — Edit Markdown content → page auto-refreshes, scroll preserved
- S3 — `krewire build` output has no HMR artifacts

## 8. Related Specifications

| SpecID      | KWF-209JV                              |
| --------- | ----------------------------------------------- |
| [KWF-PT8OD](./KWF-SSG-PT8OD-static-site-generator.md) | Core SSG |
| [KWF-99A63](./KWF-SSG-99A63-ssg-incremental-builds.md) | Incremental Builds (feeds HMR) |
| [KWF-D57UK](./KWF-SSG-D57UK-ssg-markdown-content-pipeline.md) | Content Pipeline |
| [KWF-0F2EB](./KWF-WEB-0F2EB-server-frontend-pipeline.md) | Shared Page Model |