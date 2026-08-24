# Specifications Index — Krewire Framework

This directory holds the formal specifications for the Krewire Framework.

Ordered by **impact-to-effort** (high impact, low effort first) and **dependency chain** (foundations first).

| SpecID    | Title                                      | Status | Impl Status | Depends On |
| --------- | ------------------------------------------ | ------ | ----------- | ---------- |
| [KWF-M8K2Q](./KWF-ARCH-M8K2Q-unified-framework-vision.md) | Unified Framework Vision — One Framework for Every Web Service Workload | Draft | — | KWF-CMBZJ |
| [KWF-CMBZJ](./KWF-META-CMBZJ-krewire-meta-framework.md) | Krewire Meta-Framework — Initial Specification | Draft | Shipped | — |
| [KWF-M07QS](./KWF-WEB-M07QS-krewire-web-framework.md) | Krewire Web Framework | Draft | Shipped | KWF-CMBZJ |
| [KWF-DF3PL](./KWF-SSG-DF3PL-file-site-pipeline.md) | File-Based Site Pipeline (`.kiw` Modules) | Draft | Shipped | KWF-M07QS, KWF-PT8OD |
| [KWF-PT8OD](./KWF-SSG-PT8OD-static-site-generator.md) | Static Site Generator | Draft | Shipped | KWF-M07QS |
| [KWF-C4087](./KWF-APP-C4087-krewire-app-framework.md) | Krewire App Framework (assembly) | Draft | Shipped | KWF-M07QS |
| [KWF-C9WLJ](./KWF-DI-C9WLJ-app-container-service-providers.md) | App Container & Service Providers | Draft | Shipped | KWF-CMBZJ |
| [KWF-CCI0N](./KWF-STRUCT-CCI0N-app-directory-structure.md) | App Project Directory Structure Standard | Draft | Shipped | KWF-CMBZJ |
| [KWF-5XJFC](./KWF-CLI-5XJFC-cli-application-model.md) | CLI Application Model | Draft | Shipped | KWF-CMBZJ |
| [KWF-PZ5JU](./KWF-CLI-PZ5JU-cli-scaffolding.md) | CLI Scaffolding | Draft | Planned | KWF-5XJFC |
| [KWF-0F2EB](./KWF-WEB-0F2EB-server-frontend-pipeline.md) | Server & Frontend Rendering Pipeline | Draft | Shipped | KWF-M07QS, KWF-PT8OD, KWF-C4087 |
| [KWF-D57UK](./KWF-SSG-D57UK-ssg-markdown-content-pipeline.md) | SSG Markdown Content Pipeline & Collections | Draft | Shipped | KWF-PT8OD |
| [KWF-DR5YU](./KWF-SSG-DR5YU-ssg-asset-pipeline.md) | SSG Asset Pipeline | Draft | Planned | KWF-PT8OD |
| [KWF-99A63](./KWF-SSG-99A63-ssg-incremental-builds.md) | SSG Incremental Builds & Dependency Graph | Draft | Planned | KWF-PT8OD, KWF-D57UK, KWF-DR5YU |
| [KWF-209JV](./KWF-SSG-209JV-ssg-live-reload-hmr.md) | SSG Live Reload & HMR | Draft | Planned | KWF-PT8OD, KWF-99A63 |
| [KWF-5ZHQV](./KWF-ARCH-5ZHQV-modular-monolith-architecture.md) | Modular Monolith Architecture Default | Draft | Shipped | KWF-C9WLJ, KWF-CCI0N |
| [KWF-0Z671](./KWF-UI-0Z671-krewire-ui-framework.md) | Krewire UI Framework | Draft | Shipped | KWF-M07QS |
| [KWF-PPUWX](./KWF-UI-PPUWX-layout-ui-system.md) | Layout & UI System | Draft | Shipped | KWF-0Z671 |
| [KWF-V0TMZ](./KWF-UI-V0TMZ-web-theming-system.md) | UI Theming System | Draft | Shipped | KWF-0Z671 |
| [KWF-230KF](./KWF-HTTP-230KF-http-api-pipeline.md) | HTTP & API Pipeline | Draft | Shipped | KWF-M07QS |
| [KWF-T4X9P](./KWF-WASM-T4X9P-wasm-client-runtime.md) | WASM Client Runtime — Go-Native Frontend | Draft | Planned | KWF-M8K2Q, KWF-0Z671, KWF-PT8OD |
| [KWF-B7N3D](./KWF-INFRA-B7N3D-cloud-provider-abstraction.md) | Cloud Provider Abstraction — Multi-Cloud Library-First IaC | Draft | Planned | KWF-M8K2Q |
| [KWF-L5H2F](./KWF-SVC-L5H2F-microservice-patterns.md) | Microservice & Worker Patterns | Draft | Planned | KWF-M8K2Q, KWF-5ZHQV |
| [KWF-F2TQC](./KWF-JS-F2TQC-js-ts-framework-integration.md) | JS/TS Framework Integration — Future Bridge | Draft | Planned | KWF-0Z671, KWF-PT8OD |
| [KWF-MFA0T](./KWF-CLI-MFA0T-cli-help-usage.md) | CLI Help & Usage | Draft | Shipped | KWF-5XJFC |
| [KWF-NPFSE](./KWF-CLI-NPFSE-cli-output-formatting.md) | CLI Output & Formatting | Draft | — | KWF-5XJFC |
| [KWF-KAKQL](./KWF-CLI-KAKQL-cli-errors-diagnostics.md) | CLI Errors & Diagnostics | Draft | Shipped | KWF-5XJFC |
| [KWF-FGNZ9](./KWF-CLI-FGNZ9-cli-configuration.md) | CLI Configuration | Draft | Planned | KWF-5XJFC |
| [KWF-TEST-M4P9Q](./KWF-TEST-M4P9Q-framework-test-helpers.md) | Framework Test Helpers — MVP (`framework/test`) | Draft | Shipped | KWL-TEST-P8M4L |
| [KWF-AST-K7Q2M](./KWF-AST-K7Q2M-assets-storage-system.md) | Static Assets, Resources & App Storage (`framework/assets`, `framework/storage`) | Draft | Shipped | KWF-M07QS, KWF-C4087 |
| [KWF-WEB-P3V8X](./KWF-WEB-P3V8X-expressive-http.md) | Expressive HTTP: Routes, Controllers, Request/Response, Middleware | Draft | Shipped | KWF-M07QS |
| [KWF-WEB-R9T4C](./KWF-WEB-R9T4C-http-security-state.md) | HTTP Security & State: Headers, CSRF, XSS, Cache, Session, Cookie | Draft | Shipped | KWF-WEB-P3V8X, KWF-AST-K7Q2M |
| [KWF-WEB-B2X7D](./KWF-WEB-B2X7D-auth-policy-gates.md) | Auth: Basic, JWT (HS256), Policy Gates (before/after) | Draft | Shipped | KWF-WEB-P3V8X, KWF-WEB-R9T4C |

## Conventions

- Each specification is stored as a single Markdown file named `{ProjectId}-{Scope}-{SpecID}-{slug}.md`.
- SpecIDs are unique, random 5-character alphanumeric codes (e.g., `KWF-CMBZJ`).
- New specifications must be added to this index when created.
- Ordering: impact-to-effort (high impact, low effort first), then dependency chain (foundations first).