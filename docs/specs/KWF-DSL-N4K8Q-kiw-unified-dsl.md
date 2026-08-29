# Specification — Kiw Unified DSL (`.kiw`)

| Field  | Value |
| ------ | ----- |
| SpecID | KWF-N4K8Q |
| Title  | Kiw Unified DSL — Seamless HTML · CSS · JS/TS · Go · Rust in One File |
| Status | Draft |
| Date   | 2026-08-24 |
| Author | Krewire Contributors |
| Domain | Framework — DSL & Authoring |

## 1. Summary

The **`.kiw` unified DSL** is Krewire's single-file component format that unifies **HTML, Markdown, scoped CSS, and JS/TS/Go/Rust** with deterministic compilation via `krewire build`. One file, one toolchain: HTML/Markdown describes structure and prose, CSS is scoped by default, `script[lang]` selects the execution tier (`server` in Go, `client` in Go via WASM `KWF-T4X9P` as primary — no need to write JS, `js/ts` slot remains when needed —, `compute` in Rust/WASM), and a shared `props` schema generates types for all languages from one frontmatter declaration. High-impact effortless design: Go is the primary frontend path, JS/TS is an opt-in escape hatch. This spec extends `KWF-DF3PL` (file-based `.kiw` modules) while preserving backward compatibility and the `KWF-M8K2Q` one-CLI promise. Markdown and frontmatter are optional conveniences within the same content pipeline — not separate tiers.

## 2. Background & Context

Today `.kiw` is minimal (`framework/dsl/kiw.go`, `framework/dsl/kiw.ts`, `KWF-DF3PL FRK-FLS-010`): optional frontmatter, an `html/template` body, and `<style>`/`<script>` extractions. It solved the `krewire.yaml`-inline anti-pattern, but three gaps remain:

- `web/ssg` renders `html/template` on the server; client interactivity and the WASM runtime (`KWF-T4X9P`, `KWF-F2TQC`) need tiered code.
- Go and Rust have no first-class blocks for server `Load`/`Action` or WASM compute.
- Prose inside `.kiw` currently requires verbose HTML or leaving `.kiw` for `content/*.md`; empty `---` boilerplate for components without frontmatter is noise.

`KWF-M8K2Q` and `KWF-T4X9P` require an **isomorphic** (SSR + hydration), **tiered** (server/client/compute), and **typed** (one schema, three languages) component model with ergonomic content authoring. `.kiw` is the natural place.

## 3. Problem Statement

- **Authors juggle 4+ files per component** (`.html`, `.css`, `.ts`, `.go`, `Cargo.toml`) with divergent props types; renames break at runtime instead of at `krewire build`.
- **No tier declaration** — the engine cannot decide whether a `script` runs on server (Go), client (TS/JS), or WASM compute (Rust).
- **No shared types** — `frontmatter.props` is untyped; each language invents its own shape.
- **Styling leaks** — global CSS without opt-in scoping regresses theming and `data-kiw-*` isolation.
- **Prose friction** — docs inside `.kiw` require HTML or context-switching to `content/*.md`; empty frontmatter boilerplate is unnecessary.
- If unsolved, the "one file, one CLI" promise collapses.

## 4. Goals & Non-Goals

### Goals

- G1 — One `.kiw` file composes HTML/Markdown + CSS + JS/TS + Go + Rust with **explicit tier** and **single typed `props`**.
- G2 — Elegant authoring that remains `gofmt`/`cargo fmt`/`tsc`-friendly and `html/template`-compatible.
- G3 — Deterministic `krewire build` that extracts, type-checks, and emits `site/` + `site/_assets/` (CSS, JS, Go-WASM, Rust-WASM) with content hashing.
- G4 — Scoped CSS by default + theme vars from `framework/ui`; global escape hatch explicit.
- G5 — SSR is always complete; mount points hydrate progressively (`hydrate="load|idle|visible"`) reusing `KWF-T4X9P` runtime.
- G6 — Seamless WASM: Go and Rust blocks compile to WebAssembly automatically (one shared Go runtime per site; per-module Rust compute) with generated glue and zero author-side toolchain scripting.

### Non-Goals

- NG1 — Replacing `html/template` with a new runtime templating language; the DSL compiles **to** it.
- NG2 — Bundling arbitrary npm/crates registries in v1; imports resolve within `go.mod`/`Cargo.toml` workspace only.
- NG3 — Supporting `lang="python"`/`"java"` in v1 (extensible slot reserved).
- NG4 — Live `cargo`/`go` hot-reload in this spec (covered by `KWF-209JV`); build is batch.
- NG5 — ORM or DB DSL; server Go blocks call existing `framework/app`/`storage` APIs.

### 4.5 Assumptions & Constraints

| ID | Assumption / Constraint | Type | Validation |
|----|-------------------------|------|------------|
| A1 | Go 1.22+, `krewire.yaml` is metadata-only (`KWF-DF3PL FRK-FLS-002`) | Assumption | `go vet` |
| A2 | Rust toolchain (`rustc`+`cargo`+`wasm32-unknown-unknown`) optional; missing → `compute` errors with actionable `ExitCodeUsage` | Assumption | `krewire build --check` |
| A3 | TS compiles via `esbuild` (vendored or via `go:embed` JS glue); no `node_modules` for hello-world | Assumption | fixture build |
| C1 | `framework/dsl` stays stdlib + `gopkg.in/yaml.v3` + `gomarkdown/markdown` + `x/net/html` only; no `framework` import | Constraint | `go vet ./framework/dsl` |
| C2 | Generated code never overwrites author files; all outputs under `.krewire/` and `site/_assets/` | Constraint | build test |
| C3 | Scoped CSS contract (`data-kiw-component="name"`) unchanged from `KWF-DF3PL FRK-FLS-013` | Constraint | `css_test.go` |

### 4.6 Glossary

| Term | Definition | Source |
|------|------------|--------|
| `.kiw` module | Single file with frontmatter + template/markdown + style + script tiers | `KWF-DF3PL` |
| Tier | Execution place: `server` (Go, SSR), `client` (JS/TS, browser), `compute` (Rust→WASM) | This spec |
| Content | The HTML body of a `.kiw` module — authored as HTML template, Markdown, or both; all contribute to `Body` and remain `html/template`-compatible | This spec |
| Mount point | Client-hydrated component instance (`hydrate="load"`/`"idle"`/`"visible"`) | `KWF-T4X9P FRK-WASM-040` |
| Props | Typed inputs declared in frontmatter, generated for Go/TS/Rust | This spec |

## 5. Requirements

### 5.1 Functional Requirements

Requirements are grouped by authoring concern. **Content** (HTML template + Markdown + components) shares the `Body` pipeline; **Style** and **Script** are distinct language tiers with separate compilation.

#### A. File Structure & Frontmatter

| ID | Requirement | Priority | RFC 2119 |
|----|-------------|----------|----------|
| FRK-DSL-001 | A `.kiw` file **MUST** be parsed as: optional `---` YAML frontmatter (if absent, `Frontmatter` is empty and `Body` is the whole source; empty `---` boilerplate **MUST NOT** be required), then a body that **MAY** contain `<template>`, `<style>`, `<script>`, and `<markdown>` blocks. Backward-compatible inputs (bare HTML + `<style>`/`<script>` without `lang`) **MUST** parse as before (`FRK-FLS-010`). | Must | MUST |
| FRK-DSL-002 | Frontmatter **MUST** support `name?`, `layout?`, `hydrate?` (`load`\|`idle`\|`visible`), `draft?`, `title`/`description`/`date`/`tags`, plus `props:` schema. Unknown keys **MUST** be preserved in `Frontmatter` map for forward compat. | Must | MUST |
| FRK-DSL-003 | `props:` **MUST** be a map `propName: { type, default?, required?, doc? }` where `type` is one of `string`/`int`/`float`/`bool`/`time`/`bytes`/`json`/`User` (Go type path allowed, e.g. `github.com/app.User`). The parser **MUST** generate `Props` structs/interfaces for Go (`type Props struct`), TypeScript (`interface Props`), and Rust (`struct Props` with `serde`). | Must | MUST |
| FRK-DSL-004 | Frontmatter `imports:` **MAY** declare `go: []string`, `ts: []string`, `rust: { crates: [] }` for explicit deps; absence **MUST** infer from `script` block `import` statements via static scan. | Should | MAY |

#### B. Content — Template, Components & Markdown

Template, component composition, and Markdown are facets of the same **content tier**. All contribute HTML to `Body` and remain `html/template`-compatible; Markdown is an optional prose alternative within the same file, not a separate tier.

| ID | Requirement | Priority | RFC 2119 |
|----|-------------|----------|----------|
| FRK-DSL-010 | Template **MUST** remain `html/template`-compatible. Authoring sugar `{expr}`, `{#if cond}…{/if}`, `{#each items as item}…{/each}`, `@event="handler"` and `bind:value` **MUST** desugar to `{{if}}`/`{{range}}` and `data-kiw-*` event markers at build time, never at runtime. Raw `{{.Props.X}}` **MUST** still work. | Must | MUST |
| FRK-DSL-011 | `<template>` block, if present, is the component markup; if absent, top-level markup outside `<style>`/`<script>`/`<markdown>` **MUST** be treated as template (backward compat). | Must | MUST |
| FRK-DSL-012 | Components invoke via `<Header title="Hi" />` or `{{component "Header" .}}`; both **MUST** resolve by filename (`components/Header.kiw` → `Header`) with props type-checked against the callee's `props:` schema. | Must | MUST |
| FRK-DSL-013 | Expression scope **MUST** expose `$props` (typed), `$state` (client), `$server` (server funcs), and `$compute` (Rust WASM exports) without polluting global. | Must | MUST |
| FRK-DSL-014 | `KiwModule` **MUST** expose `Markdown []string` (JSON `markdown`) holding raw Markdown sources in document order, mirroring `Styles`/`Scripts`. | Must | MUST |
| FRK-DSL-015 | `<markdown>...</markdown>` blocks (case-insensitive, any attributes ignored) **MUST** be extracted: raw `strings.TrimSpace(inner)` appended to `Markdown`, and the block in `Body` replaced with rendered HTML (`\n` + `strings.TrimSpace(renderedHTML)` + `\n`) via `gomarkdown` with extensions `CommonExtensions \| AutoHeadingIDs \| NoEmptyLineBeforeBlock` and `html.CommonFlags \| html.HrefTargetBlank`. Empty blocks **MUST** be removed without emitting HTML. Extraction **MUST** occur after `<style>`/`<script>` and preserve position via in-place replacement. | Must | MUST |
| FRK-DSL-016 | `kiw.ts` parser **MUST** mirror Go extraction: `KiwModule.markdown: string[]`, case-insensitive `<markdown>` regex, `markdown.push(inner.trim())`, and replace block with `\n` + `inner.trim()` + `\n` in `body` (JS-side rendering deferred). Existing `.kiw` without `<markdown>` **MUST** parse identically to before. | Must | MUST |

#### C. Style Tier (CSS)

| ID | Requirement | Priority | RFC 2119 |
|----|-------------|----------|----------|
| FRK-DSL-020 | `<style>` without attribute **MUST** be **scoped** (`[data-kiw-component="name"]` compounding per `KWF-DF3PL FRK-FLS-013`). `scoped` is the default; `<style global>` or `<style :global>` **MUST** be global; `:root` selectors **MUST** stay global. | Must | MUST |
| FRK-DSL-021 | `<style lang="css|scss">` **MUST** be supported; `scss` compiles via `framework`'s asset pipeline (`KWF-DR5YU`) without external `sass` binary (pure Go SCSS subset; full SCSS **MAY** defer to `dart-sass` if installed). | Should | MUST/SHOULD |
| FRK-DSL-022 | Theme vars from `framework/ui` (`--color-primary`, `--show-sun`) **MUST** be available in `<style>` as CSS custom properties; no per-component theme file required. | Must | MUST |
| FRK-DSL-023 | Multiple `<style>` blocks **MUST** concatenate in source order, hashed once into `site/_assets/<name>.<hash>.css` and linked from `<head>`. | Must | MUST |

#### D. Script Tiers (JS/TS · Go · Rust)

High-impact effortless: **Go is primary for client** (WASM `KWF-T4X9P`), `js/ts` is opt-in escape hatch when needed; `rust` compute opt-in.

| ID | Requirement | Priority | RFC 2119 |
|----|-------------|----------|----------|
| FRK-DSL-030 | `<script>` without `lang` **MUST** default to `lang="js"` `hydrate="load"` for backward compat. Explicit `lang="js|ts|go|rust"` **MUST** select the tier. Unknown `lang` **MUST** error with `ExitCodeUsage` and did-you-mean. `lang="go"` with `hydrate` selects **client WASM** tier — the primary frontend path without needing to write JS. | Must | MUST |
| FRK-DSL-031 | `client` tier: `<script lang="go|js|ts" hydrate="load|idle|visible">` **MUST** bundle: `js|ts` via `esbuild` to `site/_assets/<name>.<hash>.js`; `go` via `GOOS=js GOARCH=wasm` to `site/_assets/<name>.<hash>.wasm` + JS glue reusing shared `runtime.wasm` when possible. All injected with `type="module"` and mount marker `data-kiw-mount`, instantiate lazily per hydrate value; failed load logs console warning naming mount point and never blanks page or blocks siblings (`KWF-T4X9P FRK-WASM-042/043`). SSR HTML **MUST** remain complete when JS/WASM disabled (graceful degradation). JS is not required — Go WASM is sufficient. | Must | MUST |
| FRK-DSL-032 | `server` tier: `<script lang="go" server>` **MUST** compile as Go code with access to `context.Context`, `Props`, and `framework/*` imports. It **MUST** expose `Load`, `Action`, or `Handler` funcs that the SSR pipeline calls to populate props; errors propagate to build diagnostics. | Must | MUST |
| FRK-DSL-033 | `compute` tier: `<script lang="rust" compute>` **MUST** compile via `cargo` to `wasm32-unknown-unknown` (`site/_assets/<name>.<hash>.wasm`). Rust **MUST** use `#[kiw::export]` (proc-macro shim) to declare exports callable from Go (`$compute`) and TS (`$compute`). Compilation runs inside `krewire build` — authors never invoke cargo/go manually (G6); the Go client runtime (`KWF-T4X9P`) is likewise emitted once per site as a single shared `runtime.<hash>.wasm`, not per component. Failure without toolchain **MUST** be actionable (install hint). | Must | MUST |
| FRK-DSL-034 | Cross-tier calls **MUST** be code-generated: Go server funcs callable from TS via typed `fetch` RPC (`/__kiw/actions/<name>`), Rust exports callable from Go via WASM import and from TS via `wasm-bindgen`-like glue (generated `.js` next to `.wasm`). No manual `syscall/js` in author code. | Must | MUST |
| FRK-DSL-035 | Imports inside blocks **MUST** resolve relative to the `.kiw` file and `go.mod`/`Cargo.toml`; bare specifiers like `import { Button } from '#components'` **MUST** alias to `components/` roots without `node_modules`. | Should | MUST |
| FRK-DSL-036 | Each `script` block **MUST** be `gofmt`/`rustfmt`/`prettier`-checkable in isolation; `krewire fmt` **MUST** format `.kiw` by formatting each block with its native formatter and re-emitting the file. | Should | SHOULD |

#### E. Build, Types & Toolchain

| ID | Requirement | Priority | RFC 2119 |
|----|-------------|----------|----------|
| FRK-DSL-040 | `krewire build` **MUST** run DSL pipeline before `ssg`/`runtime`: `ParseKiw → ExtractBlocks → GenerateTypes → Compile(server|client|compute) → HashAssets → LinkHead`. `krewire dev` **MUST** watch `.kiw` and re-run affected tier only. | Must | MUST |
| FRK-DSL-041 | Generated types **MUST** live under `.krewire/gen/kiw/<path>/` (ignored by `git`), never in `pages/`/`components/`. Go types **MUST** be `package kiw` with `Props` struct; TS under `gen/kiw.d.ts`; Rust under `gen/props.rs`. | Must | MUST |
| FRK-DSL-042 | Props defaults **MUST** apply at SSR when caller omits a field; `required: true` **MUST** error at build if not provided via frontmatter or caller. | Must | MUST |
| FRK-DSL-043 | Build **MUST** be hermetic: given same `pages/`+`components/`+`krewire.yaml`, output hash **MUST** be stable; unchanged `.kiw` **MUST NOT** invalidate unrelated assets (content hashing per block). | Must | MUST |
| FRK-DSL-044 | Diagnostics **MUST** map to source locations (`pages/counter.kiw:12:4: go vet: ...`), return `ExitCodeUsage` (2) for author errors and `ExitCodeFailure` (1) for toolchain failures, via `libs/core`. | Must | MUST |
| FRK-DSL-045 | Missing toolchains are detected **before** compilation (`GOOS=js` stdlib target is always present; Rust needs `rustup target add wasm32-unknown-unknown`) and reported with install instructions, exiting `ExitCodeUsage`; `krewire build --check` **SHOULD** report toolchain readiness without building. | Must | MUST |

### 5.2 Non-Functional Requirements

| ID | Category | Requirement |
|----|----------|-------------|
| NFR1 | Performance | Hello-world `.kiw` (one template + one scoped style + one `ts` mount point + one `go` server + one `rust` compute) builds in <2s cold, <200ms warm (incremental) on CI fixture |
| NFR2 | Size | Rust `compute` hello-world WASM ≤ 50KB gzipped; TS mount JS ≤ 30KB gzipped (esbuild minify) |
| NFR3 | Quality Gates | `gofmt -l .` empty, `go vet ./...` clean, `go test ./...` passes in `framework` and `krewire`; `cargo check`/`cargo fmt --check` pass when Rust blocks present |
| NFR4 | Compatibility | Existing `.kiw` files (no `lang`, bare `<style>`/`<script>`) **MUST** build without changes |
| NFR5 | Security | No `unsafe` in `framework/dsl`; generated server code runs with `html/template` auto-escaping; `compute` WASM sandboxed (no FS/net unless explicitly imported) |
| NFR6 | Portability | Build works on `linux/amd64` without Docker; Rust `compute` requires `wasm32-unknown-unknown` target only when `compute` blocks exist |

## 6. Detailed Design / Proposal

### 6.1 Architecture

High-impact effortless: client is Go WASM by default; `js/ts` optional.

```
pages/counter.kiw
  ┌─ frontmatter (YAML) ─┐
  │ props: {initial:int} │──► .krewire/gen/kiw/counter/props.{go,ts,rs}
  ├─ <template> (HTML) ──┤──► html/template + VDOM (KWF-T4X9P)
  ├─ <style scoped> ─────┤──► ui.Theme scoping → site/_assets/counter.<hash>.css
  ├─ <script lang=go server> ──► Go package .krewire/gen/kiw/counter/server.go → SSR
  ├─ <script lang=go hydrate="load"> ──► GOOS=js GOARCH=wasm → site/_assets/counter.<hash>.wasm (primary, no JS needed)
  ├─ <script lang=ts hydrate="load"> ──► esbuild → site/_assets/counter.<hash>.js (opt-in when JS needed)
  └─ <script lang=rust compute> ──► cargo build --target wasm32 → site/_assets/counter.<hash>.wasm
                                          ↕ generated bindings
                                    Go WASM import + TS glue
  Note: <markdown> prose, when present, is rendered to HTML and injected into Body
        alongside <template> — same content pipeline, no separate tier.
```

Module: `framework/dsl` owns parsing/codegen (no `framework` import); `framework/web/ssg` consumes `*KiwModule` → `Site`; `framework/runtime` consumes mount points; `kiw/internal/commands` drives `krewire build` dispatch.

Dependencies: `framework/dsl` → `libs/core` (Kind/ExitCode) + `libs/config` (props validation) + `gopkg.in/yaml.v3` + `gomarkdown/markdown`; `framework/web/ssg` → `dsl`; `runtime` → `dsl` (types).

### 6.2 API Design

```go
// framework/dsl — extended
type KiwModule struct {
  Frontmatter map[string]any `json:"frontmatter"`
  Props       PropsSchema    `json:"props"`      // parsed from frontmatter.props
  Body        string         `json:"body"`       // template markup (desugared, markdown rendered)
  Template    string         `json:"template"`   // <template> extraction, or Body fallback
  Styles      []StyleBlock   `json:"styles"`     // { Lang, Scoped, Content, Hash }
  Scripts     []ScriptBlock  `json:"scripts"`    // { Lang, Tier, Hydrate, Content, Hash }
  Markdown    []string       `json:"markdown"`   // raw markdown sources, document order (optional)
  Raw         string         `json:"-"`
  Path        string         `json:"path"`
}
type PropsSchema struct {
  Fields []PropField `json:"fields"`
}
type PropField struct {
  Name string `json:"name"`
  Type string `json:"type"` // "string"|"int"|"github.com/x.Y"
  Default any  `json:"default,omitempty"`
  Required bool `json:"required"`
  Doc string   `json:"doc,omitempty"`
}
type StyleBlock struct { Lang string; Scoped bool; Content string }
type ScriptBlock struct { Lang string; Tier string; Hydrate string; Content string }

func ParseKiw(src string) (*KiwModule, error)
func ParseKiwFile(path string) (*KiwModule, error)
func GenerateTypes(mod *KiwModule, outDir string) error // props.{go,ts,rs}
func DesugarTemplate(src string) (string, error) // {x}→{{.x}}, @click, {#if}
func ScopeCSS(css, componentName string) (string, error) // reuses web/ssg/css.go
```

Props Go generation:

```go
// .krewire/gen/kiw/counter/props.go
package kiw

type CounterProps struct {
  Initial int `json:"initial" validate:"required"`
}
```

### 6.3 File Syntax — Author-Facing

Canonical `.kiw` (all blocks optional, order-independent, but `template` before scripts is idiomatic):

```kiw
---
name: Counter          # optional, defaults to filename
description: Click counter
hydrate: load        # load | idle | visible | false (default server-only)
props:
  initial:
    type: int
    default: 0
  label: { type: string, default: "Count" }
layout: Base
---

<template>
  <div class="counter">
    <span>{label}: {count}</span>
    <button @click="increment" class="btn">+1</button>
    {#if count > 10}
      <Badge>Hot!</Badge>
    {/if}
  </div>
</template>

<style scoped>
  .counter { display: flex; gap: 1rem; }
  .btn { background: var(--color-primary); }
</style>

<script lang="go" server>
package counter

import "context"

func Load(ctx context.Context, p CounterProps) (CounterProps, error) {
  // enrich props on server, e.g. from DB
  return p, nil
}
</script>

<script lang="go" hydrate="load">
  var count = $props.initial
  func increment() { count += 1 }
</script>

<script lang="ts" hydrate="load">
  // opt-in: only when you need JS interop
  let count = $props.initial
  function increment() { count += 1; $compute.heavy(count) }
</script>

<script lang="rust" compute>
  use kiw::prelude::*;
  #[kiw::export]
  pub fn heavy(n: i32) -> i32 { n * n }
</script>
```

For prose-heavy pages (e.g., docs), Markdown can be used directly inside `.kiw` without leaving the file:

```kiw
---
title: Getting Started
layout: Docs
---

<markdown>
# Getting Started

```bash
kiw new my-site --site
kiw build
kiw serve
```

Krewire builds from `pages/*.kiw` — file-based routing, no config.
</markdown>
```

Equivalent minimal (backward compat, no frontmatter boilerplate needed):

```kiw
<h1>{{.Title}}</h1>
<style>h1{color:red}</style>
<script>console.log(1)</script>
```

Reactivity sugar desugars at build:

| Sugar | Desugars to |
|-------|-------------|
| `{expr}` | `{{.expr}}` (escaped) / `{! expr !}` raw |
| `{#if c}…{/if}` | `{{if .c}}…{{end}}` |
| `{#each items as item}…{/each}` | `{{range .items}}…{{end}}` |
| `@click="fn"` | `data-kiw-on="click:fn"` + client listener |
| `bind:value="x"` | two-way binding via generated TS setter |
| `$props.x` | typed prop access |
| `$compute.heavy` | WASM import binding |

### 6.4 Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| A — Keep `.kiw` HTML-only, put Go/Rust in separate `*.go`/`*.rs` files | Simple parser; Go tooling native | Props drift, 4 files per component, no single-file elegance | Violates G1/G2 |
| B — Use Svelte/Vue SFC verbatim (`<script setup>`) | Familiar to frontend devs | JS-centric, Go/Rust second-class, requires Vite | Breaks Go-first, `KWF-M8K2Q` |
| C — Rust via `cgo` not WASM | Fast native | Not portable, breaks WASM runtime `KWF-T4X9P`, needs CGO | Rejected for portability (NFR6) |
| D — This proposal: tiered `<script lang>` | One file, explicit tier, typed props, WASM compute, Markdown as content alternative, backward compat | New desugar step | **Chosen** — impact high, effort medium |

Decision uses impact-to-effort ordering: foundations (parser + types) before bundles.

### 6.5 System Context & Diagrams

```
framework/dsl (parse, desugar, scope, codegen)
      ↑           ↑           ↑
web/ssg ────────┘           │
runtime (VDOM/hydration) ───┘
      ↑
kiw build (dispatch: dsl → ssg → runtime → assets)
```

Build sequence:

```
krewire build
  ├─ Load .kiw files (dsl.ParseKiwFile — frontmatter optional, markdown rendered)
  ├─ GenerateTypes (.krewire/gen/kiw/**/props.{go,ts,rs})
  ├─ Compile server Go (go vet + embed)
  ├─ Compile Rust compute (cargo build --target wasm32) — if present
  ├─ Bundle client TS/JS (esbuild) — if a mount exists
  ├─ Scope CSS (dsl.ScopeCSS)
  ├─ Render SSR (ssg + runtime VDOM)
  └─ Emit site/ + site/_assets/*.<hash>.(css|js|wasm)
```

### 6.6 Cost & Performance

| Aspect | Estimate | Notes |
|--------|----------|-------|
| Dev cost | M (2–3 weeks) | Parser 3d, types 3d, desugar 2d, Go/Rust/TS pipelines 5d, tests/docs 3d |
| Runtime cost | Zero when unused | `framework/dsl` only linked when `.kiw` present; hello-world without `rust` has no `cargo` dep |
| Build perf | <2s cold, <200ms warm | Per-block hashing, incremental tier |

### 6.7 Security, Privacy & Compliance

- Secrets: server Go blocks **MUST NOT** inline secrets; use `secret.Ref` resolved at `krewire deploy` (`KWF-B7N3D FRK-INFRA-060`).
- XSS: all `{expr}` escapes via `html/template`; raw `{! expr !}` requires explicit opt-in and lints; Markdown rendered via `gomarkdown` is HTML-escaped by `html/template` unless marked `template.HTML`.
- WASM sandbox: Rust `compute` has no FS/net/host unless `imports:` declares `wasi` — denied by default.
- Supply chain: generated Rust crate pins `kiw` shim via `Cargo.toml` workspace member, `cargo audit` in CI.

### 6.8 Accessibility & Internationalization

- Template output **MUST** preserve semantic tags and `label` associations before hydration (NFR from `KWF-T4X9P`).
- i18n: Language in specs is English per `AGENTS.md`; `props` doc strings **MAY** be Indonesian.

### 6.9 Observability

- Build logs via `log/slog` with `trace_id` when OTel enabled (`KWF-L5H2F`).
- `krewire build --verbose` prints per-tier timings and hashes.

## 7. Dependencies & Impact

- **Depends On:** `KWF-DF3PL` (file pipeline), `KWF-PT8OD` (SSG), `KWF-0Z671` (ui/theme), `KWL-CORE-K1N2Q` (Kind/ExitCode)
- **Impacts:** `framework/dsl` (parser, types, desugar, scope), `framework/web/ssg` (consume `KiwModule.Props/Scripts/Markdown`), `framework/runtime` (mount points + WASM compute loader), `framework/ui` (theme vars), `kiw/internal/commands` (`build` tier dispatch + `fmt`), `guild` (template for `.kiw` scaffolding)
- **Migration:** Legacy `.kiw` (no `lang`, no `<markdown>`, frontmatter-free components) builds unchanged; `krewire fmt` migrates formatting only.

## 8. Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Go/Rust type drift across tiers | High | Single `props:` schema → generated code; `go vet` + `tsc --noEmit` + `cargo check` in `krewire build` |
| `cargo` not installed breaks build | Medium | `krewire build --check` reports missing toolchain early; `compute` tier is opt-in |
| Desugar breaks `html/template` escaping | High | Desugar emits `{{}}` only; no string concat; golden tests for XSS |
| Bundle size bloat (JS + 2 WASM) | Medium | Content hashing, tree-shake TS, Rust `opt-level="z"` + `wasm-opt` when available |
| SpecID collision | Low | Random 5-char, check `docs/specs/index.md` |
| Markdown inside template confuses parser | Low | Extraction after `<style>`/`<script>`; position-preserving replacement; no nesting |

## 9. Testing & Verification Plan

- **Unit (`framework/dsl`):** `ParseKiw` frontmatter optional (no `---` required) + `Markdown` extraction/rendering + props schema + template desugar + `ScopeCSS`; table-driven golden tests for `{expr}`/`{#if}`/`@click` → `{{}}` and `<markdown># Hi</markdown>` → `<h1>` (FRK-DSL-010/014/020).
- **Types:** `GenerateTypes` fixture produces `props.go` that `go vet` passes, `props.ts` that `tsc --noEmit` passes, `props.rs` that `cargo check` passes.
- **Integration (`framework/web/ssg`):** fixture `pages/counter.kiw` (go+ts+rust+markdown) builds to `site/counter.html` with scoped CSS link, `site/_assets/counter.<hash>.{js,wasm,css}`; `curl` output readable without JS (SSR parity `KWF-T4X9P` NFR1).
- **Tier isolation:** fixture without `rust` block builds without `cargo`; without `client` block emits no JS; without `<markdown>` parses identically to before.
- **Spec traceability:** Each `Must` row has a test `// Tests for KWF-N4K8Q FRK-DSL-xxx` in `framework/dsl`, `framework/web/ssg`, and `kiw` fixture.
- **Gates:** `gofmt -l .` empty, `go vet ./...` clean, `go test ./...` pass (including `dsl`), `krewire build` fixture, `cargo fmt --check` when `compute` present.

## 10. Rollout & Operations

- **Phase:** Phase 1 (WASM runtime) + Phase 4 (DX unification) per `internal/docs/project-vision.md`; impact high, effort medium → after `KWF-T4X9P` VDOM merge.
- **Rollout steps:** Spec draft → review (arch-guard) → `framework/dsl` parser + types (+ Markdown) → `ssg`/`runtime` integration → `kiw build` tier dispatch → `gofmt`/`go vet`/`go test`+`cargo check`+`tsc` → `push`+`tag` `framework v0.1.0` → bump `kiw` `go.mod` → rebuild `bin/kiw` → update `docs/specs/index.md` Impl Status `Planned`→`Shipped`.
- **Timeline:** Week 1 spec review, Week 2 `dsl` + `ssg`, Week 3 `runtime` + `kiw` + docs sync (M).
- **Monitoring:** `krewire build --verbose` timing; CI fixture size budget (NFR2).
- **Rollback:** `git revert` single spec commit; legacy `.kiw` still builds.
- **Decision Log:** `2026-08-24: chose tiered <script lang> over separate files (decider: Krewire Contributors) — impact-to-effort: one file beats four.`<br>`2026-08-24: Markdown as content-tier alternative — not a separate language tier (decider: Krewire Contributors) — prose belongs to Body pipeline, unlike Style/Script which are distinct compilations.`

## 11. Open Questions

| # | Question | Owner | Resolution |
|---|----------|-------|------------|
| 1 | Should `<script lang="go" client>` (Go→WASM via `GOOS=js`) be allowed, or keep `client` as TS-only in v1? | Framework | v1 TS-only for `client`; Go→WASM client via `KWF-T4X9P` runtime, not per-component `client` |
| 2 | Rust `compute` calling Go `server` functions — via RPC or shared WASM memory? | Framework | v1 RPC (`fetch`); shared memory v2 |
| 3 | `scss` support — vendor pure Go subset or require `dart-sass`? | Framework | Pure Go subset in v1; external opt-in via `krewire.yaml ssg.scss: external` |

## 12. Success Criteria

- S1 — `pages/counter.kiw` (template + scoped style + `go server` + `ts hydrate="load"` + `rust compute` + `<markdown>` prose) builds with `krewire build` to `site/counter.html` + hashed `css`/`js`/`wasm` and passes size budgets (NFR2).
- S2 — Same fixture `curl site/counter.html` is readable without JS; button click increments via hydrated mount point in headless browser.
- S3 — `props: {initial: int}` generates `CounterProps` in `.krewire/gen/kiw/counter/props.{go,ts,rs}` and type mismatch errors at build with source location.
- S4 — `krewire fmt` formats each block with native formatter and `gofmt -l .` empty.
- S5 — Fixture without `rust` builds without `cargo`; without `client` or `<markdown>` emits no extra assets (tier isolation).
- S6 — `go doc github.com/krewire/framework/dsl` lists `ParseKiw`, `GenerateTypes`, `DesugarTemplate`.

## 13. Related Specifications

| SpecID | Title | Relationship |
|--------|-------|--------------|
| `KWF-M8K2Q` | Unified Framework Vision — One Framework for Every Web Service Workload | Source |
| `KWF-DF3PL` | File-Based Site Pipeline (`.kiw` Modules) | Extends |
| `KWF-PT8OD` | Static Site Generator | Host |
| `KWF-T4X9P` | WASM Client Runtime — Go-Native Frontend | Consumes mount points |
| `KWF-F2TQC` | JS/TS Framework Integration — Future Bridge | Superseded for `client` by this spec |
| `KWF-0Z671` | Krewire UI Framework | Theme vars for `<style>` |
| `KWF-DR5YU` | SSG Asset Pipeline | Asset hashing |
| `KWF-D57UK` | SSG Markdown Content Pipeline | Reuses `gomarkdown` parser |
| `KWF-META-CMBZJ` | Krewire Meta-Framework | Parent |
| `KWL-CORE-K1N2Q` | Core Business Rules | Exit codes / Kind |

## 14. References

- `internal/docs/project-vision.md` — unified workload matrix (9 workloads, one CLI)
- `docs/specs/index.md (framework)` — implementation matrix (Spec vs Impl Status)
- Code: `framework/dsl/kiw.go`, `framework/dsl/kiw.ts`, `framework/web/ssg/css.go`, `framework/runtime/vdom/html.go`, `framework/web/ssg/content.go`
- Astro islands architecture (prior art): https://docs.astro.build/en/concepts/islands/
- React hydration (`hydrateRoot`): https://react.dev/reference/react-dom/client/hydrateRoot
- Svelte SFC: https://svelte.dev/docs/svelte/single-file-components
- WASM (Go): https://go.dev/wiki/WebAssembly ; WASM (Rust): https://rustwasm.github.io/docs/book/
- esbuild: https://esbuild.github.io/

## 15. AI Agent Instructions

> For the AI agent that will implement this spec.

```yaml
context:
  - load: docs/specs/index.md (framework)       # check Depends On, Spec vs Impl
  - load: KWF-N4K8Q + KWF-DF3PL + KWF-T4X9P      # DSL + file pipeline + runtime
routing:
  - workload == "dsl" → skill: feature-building, agent: general
  - workload == "ssg" → skill: wasm-runtime (if hydration) or feature-building
  - multi-workload → agent: orchestrator → build
handoff:
  Task: "Kiw Unified DSL — KWF-N4K8Q"
  Kind: "site"
  Spec: "KWF-N4K8Q"
  Files: "framework/dsl/kiw.go, framework/dsl/*.ts, framework/web/ssg/*, framework/runtime/*, kiw/internal/commands/build.go"
  Gates: "gofmt -l ., go vet ./..., go test ./..., cargo check (if rust), tsc --noEmit (if ts), krewire build fixture"
gates: ["gofmt -l .", "go vet ./...", "go test ./...", "arch-guard Pass", "sync-docs In-sync"]
```

---

### Checklist Before Submitting

- [x] SpecID is random 5-char, unique, not sequenced; file name follows `KWF-DSL-N4K8Q-kiw-unified-dsl.md`
- [x] Metadata table complete, no `Version` field; `Date` is creation date
- [x] Every `Must` requirement has an ID and will trace to implementation + test
- [x] `docs/specs/index.md` and per-project `index.md` updated (next step)
- [x] Alternatives and Risks sections filled
- [x] Verification plan lists real commands (`gofmt`, `go vet`, `go test`, `cargo check`, `tsc`)

