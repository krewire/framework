# Specification — Tailwind CSS as Plugin for .kiw Sites

| Field  | Value |
| ------ | ----- |
| SpecID | KWF-T9X2K |
| Title  | Tailwind CSS as Plugin for .kiw Sites |
| Status | Draft |
| Date   | 2026-08-24 |
| Author | Krewire Contributors |
| Domain | Framework — UI & Site Pipeline |

## 1. Summary

Tailwind CSS is supported as an **optional PostCSS plugin** for `site` workloads — like in Astro (`@astrojs/tailwind`), Next.js (`tailwindcss` + `postcss.config.js`), or Vite — not as a core Krewire feature. When a site opts in by adding `tailwind.config.js` (and `assets/tailwind.css` with `@tailwind` directives), `krewire build` detects it, runs the Tailwind CLI via PostCSS, emits `site/assets/tailwind.<hash>.css`, and links it in `<head>`. Utility classes then work directly in `.kiw` HTML/Markdown without extra syntax.

## 2. Background & Context

- `KWF-DSL-N4K8Q` defines `.kiw` authoring (HTML/Markdown + scoped CSS + tiered scripts) and `KWF-PT8OD` the file-based `site` pipeline (`pages/**/*.kiw` → `site/`). `KWF-0Z671` provides theme vars (`--color-primary` `#00c853`) but no utility framework.
- Landing `krewire/krewire` currently uses hand-written scoped CSS for hero/cards. Adopting Tailwind as a plugin reduces custom CSS and validates the `site` pipeline with a real-world utility framework, while keeping the core framework agnostic — Tailwind is one of many possible PostCSS plugins, not a built-in.
- Modern frameworks treat Tailwind as an **integration**: `npx astro add tailwind` adds `tailwind()` to `astro.config.mjs` and creates `tailwind.config.mjs`; `npx tailwindcss init -p` creates `tailwind.config.js` + `postcss.config.js` and `globals.css` with `@tailwind` directives. Krewire mirrors this: the presence of `tailwind.config.js` at the project root is the opt-in signal, not a `tailwind:` key in `krewire.yaml`.

## 3. Problem Statement

- Authors cannot use Tailwind utilities in `.kiw` without manual `npx tailwindcss` setup per project; the pipeline has no PostCSS hook or asset handling for plugins.
- Landing's hand-written CSS duplicates utilities (`flex`, `gap`, `p-4`, `rounded-xl`) that Tailwind provides.
- Without a spec, Tailwind integration would be ad-hoc (CDN vs CLI, config location, hashing, linking) and would appear as if Tailwind were part of the Krewire core, rather than an optional plugin.

## 4. Goals & Non-Goals

### Goals

- G1 — Tailwind as **plugin**: adding `tailwind.config.js` + `assets/tailwind.css` (`@tailwind base/components/utilities`) at the site root is sufficient — no `tailwind:` key in `krewire.yaml`, no per-component boilerplate.
- G2 — `krewire build` auto-detects the plugin (presence of `tailwind.config.js` or `postcss.config.js` containing `tailwindcss`), runs Tailwind CLI (`npx tailwindcss -i <input> -o <output> --minify` with `content` from the config), emits `site/assets/tailwind.<hash>.css`, and injects `<link>` after `style.css` and before `theme.css`.
- G3 — Utility classes work directly in `.kiw` HTML/Markdown: `<div class="flex gap-4">` is retained via Tailwind's content detection.
- G4 — Backward compatible: sites without `tailwind.config.js` build exactly as before (no extra asset, no Node.js requirement).

### Non-Goals

- NG1 — No Tailwind plugins or custom JIT config beyond `tailwind.config.js` in v0.1.0.
- NG2 — No `lang="tailwind"` for `<style>`; Tailwind is a site-level PostCSS plugin, not a per-component tier.
- NG3 — No runtime Tailwind CDN in production; CDN is dev-preview only.

### 4.5 Assumptions & Constraints

| ID | Assumption / Constraint | Type | Validation |
|----|-------------------------|------|------------|
| A1 | Node.js + `tailwindcss` CLI available in CI when `tailwind.config.js` exists (via `npx tailwindcss` or `node_modules/.bin/tailwindcss`) | Assumption | `krewire build --check` |
| A2 | `tailwind.config.js` `content` covers `pages/**/*.kiw`, `components/**/*.kiw`, `layouts/**/*.kiw`, `content/**/*.md` (like Astro/Next) | Assumption | fixture build |
| C1 | `framework/dsl` and `framework/web/ssg` remain Go-only; Tailwind CLI is invoked as an external PostCSS process from `kiw`/`ssg`, not embedded | Constraint | `go vet ./framework/...` |
| C2 | Tailwind output is hashed and linked deterministically, like other assets | Constraint | build test |
| C3 | Tailwind is one PostCSS plugin among many; the pipeline must not hardcode Tailwind as a core dependency | Constraint | `go vet` |

## 5. Requirements

| ID | Requirement | Priority | RFC 2119 |
|----|-------------|----------|----------|
| FRK-UI-001 | A site **MAY** opt into Tailwind by placing `tailwind.config.js` (and `assets/tailwind.css` with `@tailwind base; @tailwind components; @tailwind utilities;`) at the project root. No `tailwind:` key in `krewire.yaml` is required. Absence means Tailwind is not used. | Must | MUST |
| FRK-UI-002 | When `tailwind.config.js` exists, `krewire build` **MUST** run Tailwind CLI (`npx tailwindcss -i assets/tailwind.css -o <tmp>/tailwind.css --minify`, respecting `config` path if present) via PostCSS, write output to `site/assets/tailwind.<hash>.css`, and inject `<link rel="stylesheet" href="/assets/tailwind.<hash>.css">` into `<head>` (after `style.css`, before `theme.css`). | Must | MUST |
| FRK-UI-003 | Tailwind utility classes **MUST** be usable directly in `.kiw` HTML/Markdown without extra syntax: `<div class="flex gap-4 p-4 rounded-xl">` renders as-is and is retained via Tailwind's `content` detection. | Must | MUST |
| FRK-UI-004 | `krewire build --check` **SHOULD** report Tailwind toolchain readiness (Node.js, `tailwindcss` CLI) when `tailwind.config.js` exists, exiting `ExitCodeUsage` with install hint (`npm install -D tailwindcss`) if missing. | Should | SHOULD |
| FRK-UI-005 | Sites without `tailwind.config.js` **MUST** build identically to before (no extra asset, no extra link, no Node.js requirement). | Must | MUST |
| FRK-UI-006 | `krewire/krewire` **MAY** adopt Tailwind as a plugin: add `tailwind.config.js` (`content: ["pages/**/*.kiw","components/**/*.kiw","layouts/**/*.kiw","content/**/*.md"]`, `theme.extend.colors.primary: "var(--color-primary)"`), `assets/tailwind.css` with directives, and migrate hero/feature-grid to utilities, keeping theme vars and scoped CSS for overrides. | Should | SHOULD |

## 6. Detailed Design

### 6.1 Opt-in (like Astro/Next/Vite)

```bash
# at site root (krewire/krewire)
npm install -D tailwindcss postcss autoprefixer
npx tailwindcss init -p   # creates tailwind.config.js + postcss.config.js
```

```js
// tailwind.config.js
module.exports = {
  content: ["pages/**/*.kiw", "components/**/*.kiw", "layouts/**/*.kiw", "content/**/*.md"],
  theme: { extend: { colors: { primary: "var(--color-primary)" } } },
}
```

```css
/* assets/tailwind.css */
@tailwind base;
@tailwind components;
@tailwind utilities;
@layer base { :root { --color-primary: #00c853; } }
```

No `krewire.yaml` change — the presence of `tailwind.config.js` is the plugin signal (like `astro.config.mjs` with `tailwind()`).

### 6.2 Pipeline

```
krewire build (site)
  ├─ Load .kiw (dsl) — unchanged
  ├─ If tailwind.config.js exists at project root:
  │   ├─ Run tailwindcss CLI (npx or ./node_modules/.bin/tailwindcss)
  │   ├─ Hash output → site/assets/tailwind.<hash>.css
  │   └─ Inject link into <head>
  ├─ Scope CSS → site/assets/style.<hash>.css (existing)
  └─ Emit site/
```

The `site` pipeline treats Tailwind as one PostCSS plugin; other plugins (e.g., `autoprefixer`) work the same via `postcss.config.js` without Krewire core changes.

### 6.3 Landing Migration (example)

- `Hero.kiw`: `class="grid md:grid-cols-2 gap-10"` instead of hand-written `display:grid`
- `FeatureCard.kiw`: `class="rounded-2xl border p-5"` instead of scoped `.card`
- `tailwind.config.js` as above; `assets/tailwind.css` directives

## 7. Testing & Verification

- Unit: `web/ssg` with `tailwind.config.js` fixture builds `site/assets/tailwind.<hash>.css` and `site/index.html` contains `<link href="/assets/tailwind.`, without it no link.
- Integration: `krewire/krewire` with `tailwind.config.js` + `assets/tailwind.css` `kiw build` produces styled landing and `curl` readable without JS.
- Gates: `go vet ./...`, `go test ./...`, `krewire build` fixture, `npx tailwindcss --help` when plugin present.

## 8. Related Specifications

| SpecID | Title | Relationship |
|--------|-------|--------------|
| KWF-DSL-N4K8Q | Kiw Unified DSL | Extends (style tier as plugin) |
| KWF-0Z671 | UI Framework | Theme vars |
| KWF-PT8OD | Static Site Generator | Host |

## 9. AI Agent Instructions

```yaml
context: [framework/web/ssg, krewire/krewire]
gates: ["go vet ./...", "go test ./...", "krewire build (with tailwind fixture if present)"]
```

### Checklist
- [x] SpecID random, file name `{ProjectId}-{Scope}-{SpecID}-{slug}.md`
- [x] Every Must has ID
