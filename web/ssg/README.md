# web/ssg — File-Based Static Site Generator

Build static sites from `.kiw` files (see [`framework/dsl`](../../dsl)) — Astro-inspired, Go-first.

## Layout

```
project/
├── krewire.yaml        # metadata only: title, description, theme, output
├── pages/              # one .kiw per route (file-based, extensionless)
│   ├── index.kiw       # → /
│   ├── about.kiw       # → /about
│   └── blog/
│       ├── index.kiw   # → /blog  (listing via .Collections)
│       └── [slug].kiw  # → /blog/<slug> for each content/blog/*.md
├── components/*.kiw    # reusable; invoked with {{component "Name"}}
├── layouts/*.kiw       # shells with {{.Content}}; default Base.kiw
├── content/<name>/*.md # collections (frontmatter + Markdown)
└── public/             # copied verbatim; overrides generated assets
```

## Example

`pages/index.kiw`:

```kiw
---
title: Home
layout: Base
---

<section class="hero">
  <h1>{{.Title}}</h1>
  {{component "Badge" "fast"}}
</section>

<style>
.hero h1 { color: var(--primary); }
</style>

<script>console.log("loaded")</script>
```

`layouts/Base.kiw`:

```kiw
<html>
<head><title>{{.Title}}</title></head>
<body>{{.Content}}</body></html>
<style>body { margin: 0; }</style>
```

## Build

```bash
kiw build          # detects pages/*.kiw → file mode
kiw serve          # preview site/
```

Programmatic API:

```go
site, err := ssg.LoadFromDir(root) // file-based (.kiw)
created, err := site.Build(outDir)

// or declarative config:
created, err := ssg.BuildFromConfig(cfg, outDir)
```

## Data available in templates

- `.Title`, `.Description`, `.Theme` — from `krewire.yaml`
- `.Collections.<name>` — list of collection items (`Title`, `Date`, `Permalink`, `Content`, `Slug`, ...)
- `.Content`, `.Page` — for `[slug].kiw` pages, the matched item's content and metadata
- Frontmatter fields are merged into page data

## Rules

- Routes are file-based and extensionless; each emits a sibling `.html` file: `/` → `index.html`, `/about` → `about.html`, `/docs/quickstart` → `docs/quickstart.html`
- Scoped CSS: `<style>` blocks rewrite under `[data-kiw-component="name"]`; `:root` stays global
- Files starting with `_` are partials — never routed
- `public/` overrides machine-generated assets at the same path
- Draft/future-dated content items are excluded from builds
- Books stay in mdbind (`manuscript/`) — this package does not handle them
