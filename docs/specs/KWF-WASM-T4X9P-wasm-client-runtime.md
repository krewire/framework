# Specification — WASM Client Runtime

| Field  | Value                           |
| ------ | ------------------------------- |
| SpecID | KWF-T4X9P                       |
| Title  | WASM Client Runtime — Go-Native Frontend |
| Status | Draft                           |
| Date   | 2026-08-21                      |
| Author | Krewire Contributors             |
| Domain | Framework — Frontend Runtime    |

## 1. Context

Krewire's current rendering story is server-side: `framework/web/ssg` and
`mdbind` produce HTML with scoped CSS (KWF-0Z671, KWF-PT8OD), `framework/ui`
provides themes and components, and any browser interactivity must go through a
future JS/TS bridge (KWF-F2TQC). That bridge splits the component model and
the toolchain by language — SSR in Go, interactivity in JS.

The unified framework (KWF-M8K2Q G2) requires a Go-native client runtime:
components renderable on both server and browser, initial content visible
without JavaScript, interactivity hydrated progressively — progressive
hydration with Go as the single language. v1 targets the standard Go
WASM backend (`GOOS=js GOARCH=wasm`), not TinyGo, to keep the full standard
library and simplify tooling.

## 2. Problem Statement

- Server-rendered sites are static once delivered; any interactivity leaks
  developers into a JS toolchain (KWF-F2TQC placeholder).
- There is no shared component model between SSR output and browser behavior,
  so hydration is impossible.
- The framework needs a path to reactive UI (state, events, layout) without
  forcing teams to adopt a second language or bundler.

## 3. Goals

- G1 — `krewire build --target=wasm` (and implicitly for `site`/`app` projects)
  produces a `.wasm` module + JS glue with the runtime embedded.
- G2 — Minimal `syscall/js` DOM bridge: element creation, attribute updates,
  event listeners, `requestAnimationFrame` loop.
- G3 — Virtual DOM diff/patch usable for both server string rendering and
  client DOM reconciliation.
- G4 — Component model with lifecycle (`OnMount`, `OnUnmount`) and composable
  hooks (`useState`, `useEffect`, `useMemo` via Go generics) shared by server
  and client renders.
- G5 — Progressive hydration: `hydrate="load"` / `"idle"` / `"visible"`
  mount points connect SSR HTML to client components without double rendering.
- G6 — Starter widget set sufficient for a product dashboard: layout and basic
  input/display widgets.
- G7 — Theme and scoped CSS from `framework/ui` compile to a shape both the
  SSG and the runtime consume, with no visual mismatch between SSR and hydrate.

## 4. Non-Goals

- NG1 — TinyGo target; the first release uses only standard Go WASM.
- NG2 — CSS-in-JS runtime authoring; authoring stays as scoped CSS and
  `framework/ui` Theme vars, compiled at build time.
- NG3 — Native desktop/mobile targets; they are future work after the web
  runtime is proven.
- NG4 — Full parity with Flutter's widget catalog; v1 ships a deliberate
  small core, extensible by users.
- NG5 — Replacing the existing SSG pipeline; the runtime integrates with
  `web/ssg`, it does not rewrite it.

## 5. Requirements

### 5.1 Build Pipeline

| ID            | Requirement                                                            | Priority |
| ------------- | ---------------------------------------------------------------------- | -------- |
| FRK-WASM-001  | `runtime/build` exposes `BuildWASM(cfg) (Artifacts, error)` using     | Must     |
|               | `go build GOOS=js GOARCH=wasm` with `bytes.Buffer` log capture.         |          |
| FRK-WASM-002  | `krewire build` detects components with client directives and runs WASM | Must     |
|               | build before emitting `site/`; outputs `site/_assets/runtime.*` pair.   |          |
| FRK-WASM-003  | Assets are content-hashed and linked from the generated `<head>`;     | Must     |
|               | unchanged runtime is cache-stable across SSG rebuilds.                  |          |
| FRK-WASM-004  | Build fails with actionable diagnostics (Go toolchain location, `go     | Must     |
|               | version` mismatch) and returns exit code 2 on usage errors.             |          |

### 5.2 DOM & Event Bridge

| ID            | Requirement                                                            | Priority |
| ------------- | ---------------------------------------------------------------------- | -------- |
| FRK-WASM-010  | `runtime/js` provides thin wrappers over `syscall/js`: `Element`,      | Must     |
|               | `CreateElement`, `SetAttr`, `Append`, `Remove`, `AddEventListener`.    |          |
| FRK-WASM-011  | Synthetic event system normalizes `click`, `input`, `submit` across    | Should   |
|               | browsers and supports delegation from a root listener.                  |          |
| FRK-WASM-012  | `requestAnimationFrame` loop drives scheduled re-renders, batched per   | Must     |
|               | frame; manual `Flush()` remains available for tests.                    |          |

### 5.3 Virtual DOM

| ID            | Requirement                                                            | Priority |
| ------------- | ---------------------------------------------------------------------- | -------- |
| FRK-WASM-020  | `runtime/vdom.VNode` holds `Tag`, `Props`, `Children`, optional `Key`  | Must     |
|               | and `ComponentType` discriminator for hosted components.                |          |
| FRK-WASM-021  | `Diff(old, next) []Patch` handles keyed children in O(n) and emits    | Must     |
|               | `Insert`, `Remove`, `UpdateProps`, `UpdateText`, `Replace`.            |          |
| FRK-WASM-022  | Server `RenderHTML(tree) string` and client `PatchDOM(patches)` share | Must     |
|               | the same prop normalization rules.                                      |          |
| FRK-WASM-023  | `go test ./runtime/vdom` covers keyed reorders, nil children, and      | Must     |
|               | prop diffing with >80% coverage of `diff.go`.                           |          |

### 5.4 Component Model & Hooks

| ID            | Requirement                                                            | Priority |
| ------------- | ---------------------------------------------------------------------- | -------- |
| FRK-WASM-030  | `runtime/component.Component` interface: `Render() VNode`; optional     | Must     |
|               | `OnMount`, `OnUnmount`, `ShouldUpdate` hooks.                           |          |
| FRK-WASM-031  | Generic hooks `UseState[T]`, `UseEffect`, `UseRef[T]`, `UseMemo[T]`    | Must     |
|               | tie into the per-component fiber and enforce stable call order.         |          |
| FRK-WASM-032  | Component registry `Register(name, factory)` enables SSR lookup and     | Must     |
|               | client hydration by the same `data-kiw-component` key.                   |          |
| FRK-WASM-033  | Violating hook rules (conditional call) panics in tests and is          | Should   |
|               | diagnosed via `go vet`-friendly function naming.                        |          |

### 5.5 Hydration & Mount Points

| ID            | Requirement                                                            | Priority |
| ------------- | ---------------------------------------------------------------------- | -------- |
| FRK-WASM-040  | SSG recognizes `hydrate="load"`, `hydrate="idle"`, `hydrate="visible"` | Must     |
|               | on a component instance and emits `data-kiw-mount` markers + props JSON. |          |
| FRK-WASM-041  | Client boot scans markers, instantiates matching components, and        | Must     |
|               | attaches listeners without re-rendering SSR text nodes (text parity).   |          |
| FRK-WASM-042  | Content is readable and routes navigable before WASM loads (graceful   | Must     |
|               | degradation): SSR HTML is always complete.                              |          |
| FRK-WASM-043  | When hydration text mismatches, a console warning identifies the mount  | Should   |
|               | point name and prop key without crashing the page.                      |          |

### 5.6 Starter Widgets

v1 catalog (installed under `runtime/widgets`):

| ID            | Requirement                                                            | Priority |
| ------------- | ---------------------------------------------------------------------- | -------- |
| FRK-WASM-050  | Layout: `Container`, `Row`, `Column`, `Stack`, `Expanded`, `SizedBox`. | Must     |
| FRK-WASM-051  | Display: `Text`, `Image`, `Icon`.                                      | Must     |
| FRK-WASM-052  | Inputs: `Button`, `Input`, `Checkbox`, `Switch`.                        | Must     |
| FRK-WASM-053  | Structure: `Scaffold`, `AppBar`, `Card`.                                | Should   |
| FRK-WASM-054  | Lists: `ListView`, `ListTile` (virtualization deferred to v2).         | Should   |
| FRK-WASM-055  | Each widget ships with SSR rendering and client behavior covered by a   | Must     |
|               | paired test (golden HTML + simulated click/input).                      |          |

### 5.7 Theming & Scoped CSS

| ID            | Requirement                                                            | Priority |
| ------------- | ---------------------------------------------------------------------- | -------- |
| FRK-WASM-060  | `framework/ui` Theme vars map to CSS custom properties under             | Must     |
|               | `data-kiw-theme`; SSG injects them once in `<head>`, runtime reuses.    |          |
| FRK-WASM-061  | SSG scoped CSS (`data-kiw-component` compounding) and runtime-injected   | Must     |
|               | styles produce identical computed styles for the same component name.    |          |
| FRK-WASM-062  | Route-level theme overrides propagate to hydrated mounts via inherited  | Should   |
|               | CSS vars, requiring no per-mount stylesheet.                            |          |

### 5.8 Size & Performance Budgets

| ID            | Requirement                                                            | Priority |
| ------------- | ---------------------------------------------------------------------- | -------- |
| FRK-WASM-070  | Hello-world app with one mount point is ≤ 800 KB gzipped total (`.wasm` +   | Must     |
|               | JS glue) measured on the CI fixture.                                    |          |
| FRK-WASM-071  | First hydration completes within 500ms after WASM instantiate on the    | Should   |
|               | CI fixture (Chrome headless), excluding network.                        |          |
| FRK-WASM-072  | Server render path allocates no WASM and is testable with `go vet`.    | Must     |

## 6. Non-Functional Requirements

- NFR1 — **SSR/client parity** is deterministic: `RenderHTML(Comp(props))`
  equals the server-emitted mount HTML after prop canonicalization.
- NFR2 — **Accessibility**: SSR output preserves semantic tags and label
  associations before hydration; client hydration does not regress AX.
- NFR3 — **Quality gates**: `gofmt`, `go vet ./...`, `go test ./...` in
  `framework` and `krewire` pass, the latter with fixture builds.
- NFR4 — **Storage**: theme preference (`localStorage "krewire-theme"`) is
  respected by both the SSG `<head>` script and the runtime toggle.

## 7. Success Criteria

- S1 — A counter mount point built from `ListView` + `Button` + `useState` renders
  via SSR, hydrates, and increments on click in a headless browser test.
- S2 — `krewire build` on the fixture produces `site/_assets/runtime.*.wasm`
  and a `site/index.html` whose `curl` output is readable without JS.
- S3 — Theme variables from `framework/ui` affect both SSR and hydrated
  rendering and the toggle persists across reloads.
- S4 — `go test ./framework/runtime/...` demonstrates VDOM, component, and
  hydration coverage, and `gofmt -l .` is empty.

## 8. Related Specifications

| SpecID    | Title                                               |
| --------- | --------------------------------------------------- |
| [KWF-M8K2Q](./KWF-ARCH-M8K2Q-unified-framework-vision.md) | Unified framework vision (parent) |
| [KWF-PT8OD](./KWF-SSG-PT8OD-static-site-generator.md)    | Static site generator (SSR host)  |
| [KWF-0Z671](./KWF-UI-0Z671-krewire-ui-framework.md)       | UI framework & theming            |
| [KWF-DR5YU](./KWF-SSG-DR5YU-ssg-asset-pipeline.md)       | SSG asset pipeline                |
| [KWF-F2TQC](./KWF-JS-F2TQC-js-ts-framework-integration.md) | JS/TS bridge (superseded by this spec for client interactivity) |

## 9. References

- Go WebAssembly: https://go.dev/wiki/WebAssembly
- React hydration (`hydrateRoot`): https://react.dev/reference/react-dom/client/hydrateRoot
- Astro islands architecture (prior art for progressive hydration): https://docs.astro.build/en/concepts/islands/
- Flutter rendering pipeline: https://docs.flutter.dev/resources/architecture
- HMR/VDOM tradeoff analysis: https://crafts.astro.build/
