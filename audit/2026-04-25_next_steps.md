# Next Steps

## P0 - Must fix before moving forward

### 1. Resolve database reality
- Why it matters: project direction says PostgreSQL, runtime is SQLite
- Affected files/modules: `backend/internal/app/config.go`, `backend/internal/app/server.go`, `backend/internal/app/store.go`, `.env`, `README.md`
- Estimated complexity: large
- Recommended Codex prompt topic: migrate backend from SQLite bootstrap to real PostgreSQL connection and tracked migrations

### 2. Stabilize backend contracts
- Why it matters: mixed response and error shapes create fragile frontend wiring
- Affected files/modules: `backend/internal/app/server.go`, frontend stores
- Estimated complexity: medium
- Recommended Codex prompt topic: unify API response contract and error handling across all site-scoped admin endpoints

### 3. Make editor truth model consistent
- Why it matters: UI advertises multi-file editing but backend only saves `index.html`
- Affected files/modules: `frontend/src/views/EditorView.vue`, `frontend/src/stores/editor.js`, backend workspace handlers
- Estimated complexity: large
- Recommended Codex prompt topic: implement real multi-file workspace persistence and asset storage without breaking publish

## P1 - Needed for template/data embedding

### 4. Add explicit publish render pipeline
- Why it matters: template embedding needs a render stage between workspace source and published output
- Affected files/modules: publish handler, filesystem helpers, future renderer package
- Estimated complexity: large
- Recommended Codex prompt topic: design publish-time template renderer for site data tags and loops

### 5. Define safe template language and escaping rules
- Why it matters: `{{ site.name }}` and loops require safe parsing and escaping
- Affected files/modules: new renderer/template package, publish flow
- Estimated complexity: large
- Recommended Codex prompt topic: add safe HTML template tag parser for site, product, post, and subscription data

### 6. Add data source contracts for rendering
- Why it matters: renderer needs stable inputs for products, posts, plans, and site metadata
- Affected files/modules: backend domain queries, render context builder
- Estimated complexity: medium
- Recommended Codex prompt topic: build site-scoped render context for products, posts, and subscription plans

## P2 - Needed for better UX

### 7. Replace placeholder admin interactions with full CRUD flows
- Why it matters: dashboards still mix real and placeholder behavior
- Affected files/modules: `frontend/src/views/SiteDashboardView.vue`, frontend stores, backend admin endpoints
- Estimated complexity: medium
- Recommended Codex prompt topic: complete real CRUD wiring for secondary tabs in ecommerce, blogs, and subscriptions

### 8. Introduce Monaco or real code editor integration
- Why it matters: current textarea editor limits usability and future template editing experience
- Affected files/modules: editor view, editor store, frontend build deps
- Estimated complexity: medium
- Recommended Codex prompt topic: replace textarea editor with lazy-loaded Monaco while keeping publish flow intact

### 9. Improve auth and authorization boundaries
- Why it matters: admin APIs should be site/tenant scoped securely before more data is added
- Affected files/modules: auth handling, API middleware, site queries
- Estimated complexity: medium
- Recommended Codex prompt topic: enforce auth and site ownership checks on all admin APIs

## P3 - Later features

### 10. Add embedded snippet UI in editor
- Why it matters: authors need a guided way to insert template tags later
- Affected files/modules: editor UI, future snippet panel components
- Estimated complexity: medium
- Recommended Codex prompt topic: add template snippet browser for site fields, product loops, post loops, and subscription loops

### 11. Add preview render mode for data templates
- Why it matters: authors need confidence before publish
- Affected files/modules: preview pipeline, renderer, editor preview
- Estimated complexity: large
- Recommended Codex prompt topic: add preview rendering mode that resolves template tags with sample site data

### 12. Add migration tracking and operational docs
- Why it matters: growing schema needs repeatable deployment and rollback discipline
- Affected files/modules: backend migrations, README, dev scripts
- Estimated complexity: medium
- Recommended Codex prompt topic: add schema_migrations tracking and deployment-safe migration runner
