# WebsiteBuilder Full Audit

## Executive Summary
The project currently works as an HTML-first website builder POC with a usable flow for create site -> edit HTML -> save -> publish -> open live website. The strongest part of the system is the filesystem-based publish flow. The weakest part is architecture drift: the repo and `.env` suggest a PostgreSQL-backed product platform, but the actual runtime still uses SQLite, a monolithic backend file, and partial/mixed frontend admin wiring. Ecommerce, Blogs, and Subscription Stores now exist as product surfaces, but they are not yet backed by a clean, stable foundation.

## Current Product State
- Core builder POC is functional.
- Admin UI is much broader than the original builder scope.
- Delivery is static and works for published `index.html`.
- The platform is not yet ready for safe template/data embedding.

## What Works
- Basic auth flow
- Project/site listing
- Site dashboard routing
- Editor route and editor shell
- Save of workspace `index.html`
- Publish copies workspace HTML to published HTML
- Public delivery by site slug
- Launcher script and log outputs
- Seed data for demo usage
- Basic CRUD persistence for parts of products/posts/plans

## What Is Partially Working
- Editor behaves like a multi-file IDE in UI, but backend only persists `index.html`
- Ecommerce workspace has real APIs for some sections, but not every visible action is equally complete
- Blogs workspace has real post persistence plus supporting data APIs, but some secondary flows remain shallow
- Subscription Stores surface has API-backed data but still relies on simplified assumptions
- API response formats are not fully standardized

## What Is Broken
- PostgreSQL is not actually used
- `.env`, README, and runtime behavior disagree on architecture
- Root repo contains leftover structure drift (`cmd/`, `internal/`) alongside active `backend/`
- Editor file explorer overpromises actual persistence
- No render stage exists for template/data embedding

## What Is Overbuilt
- Admin product breadth compared to current backend discipline
- Large API surface without proper domain/module separation
- UI scope for commerce/blog/subscriptions ahead of true platform stability

## What Is Missing
- Real PostgreSQL connection and migration tracking
- `schema_migrations`
- Service/repository separation
- Unified API error contract
- Real multi-file workspace persistence
- Asset persistence and delivery
- Safe template engine and publish-time rendering
- Strong auth/authorization around site-scoped admin APIs
- Snippet/template insertion UX in editor

## Scoring
- Architecture alignment score: 4/10
- POC readiness score: 7/10
- Product risk score: 8/10

## Recommended Immediate Focus
Stabilize the foundation before adding embedded data templates:
1. Align backend/database reality
2. Standardize API contracts
3. Make editor persistence truthful
4. Add a proper publish render stage only after that

---

## Backend Audit

### Go project structure
- Status: PARTIAL
- Active app is under `backend/`, but root-level residue remains.

### API route organization
- Status: PARTIAL
- Too many routes live in one server file.

### service/handler/repository separation
- Status: MISSING
- Handlers mix validation, SQL, defaults, and response shaping.

### config loading
- Status: PARTIAL
- Config exists, but active DB config is SQLite-only.

### PostgreSQL connection
- Status: BROKEN
- Runtime opens SQLite via `modernc.org/sqlite`.

### error handling
- Status: PARTIAL
- Mixed plain-string and nested-object patterns.

### logging
- Status: PARTIAL
- Startup and publish logs exist; structured request/domain logging does not.

### middleware
- Status: PARTIAL
- CORS exists. Auth enforcement is inconsistent on admin routes.

### health endpoints
- Status: OK
- `/healthz` exists.

### publish API
- Status: OK
- Reads workspace HTML, writes published HTML, verifies output, returns `public_url`.

### delivery API
- Status: OK
- Serves published HTML from filesystem by site slug.

### ecommerce APIs
- Status: PARTIAL
- Surface exists, but implementation quality and consistency vary.

### blog APIs
- Status: PARTIAL
- Many routes exist, but depth and separation are still POC-grade.

### subscription APIs
- Status: PARTIAL
- Present, but simplified and not deeply validated.

### template/data embedding readiness
- Status: BROKEN
- Delivery is static-only and there is no render phase.

### Backend Risks
- Database direction mismatch
- Growing monolithic backend
- Inconsistent contracts
- Weak domain separation
- Static-only publish pipeline

---

## Frontend Audit

### Vue project structure
- Status: PARTIAL
- Simple and workable, but important views are too large.

### routing
- Status: OK
- Core route structure is clear and working.

### Pinia stores
- Status: PARTIAL
- Domain stores exist, but some are broad aggregators over many endpoints.

### API service layer
- Status: PARTIAL
- One axios client works for POC, but domain service separation is missing.

### editor UX
- Status: PARTIAL
- Stronger visually, but still textarea-based and only partially backed by real persistence.

### file explorer UX
- Status: PARTIAL
- Good UI feel, but many actions are frontend-local.

### save/validate/publish wiring
- Status: PARTIAL
- Works for `index.html`, not for the broader workspace illusion.

### publish success modal
- Status: OK

### ecommerce workspace
- Status: PARTIAL

### blog workspace
- Status: PARTIAL

### subscription stores workspace
- Status: PARTIAL

### UI density/polish
- Status: OK

### broken interactions
- Status: PARTIAL
- Some actions are placeholders or shallow wrappers over simple backend behavior.

### mock vs real data
- Status: PARTIAL
- Better than before, but still mixed.

### future template-snippet panel readiness
- Status: MISSING

---

## Database Audit

### database bootstrap
- Status: PARTIAL
- Schema bootstrap exists in Go startup code.

### migrations
- Status: PARTIAL
- Repeat-safe schema creation exists, but no real migration framework.

### schema_migrations
- Status: MISSING

### sites table
- Status: PARTIAL
- `projects` acts as the site table.

### ecommerce tables
- Status: PARTIAL

### blog tables
- Status: PARTIAL

### subscription tables
- Status: PARTIAL

### indexes
- Status: PARTIAL
- Several exist, but approach is hand-managed and untracked.

### constraints
- Status: PARTIAL
- Some uniqueness and FK coverage exists, but many business rules remain application-level.

### seed/default data
- Status: OK

### SQLite remnants
- Status: BROKEN
- SQLite is not a remnant; it is the active DB.

### HTML source storage check
- Status: OK
- Website HTML source is not stored in DB.

### metadata-only rule
- Status: PARTIAL
- Site HTML stays on filesystem; blog content is stored in DB, which is acceptable for blog domain data.

### PostgreSQL readiness
- Status: BROKEN

### Explicit confirmation
- Website HTML source stored in DB: No
- Website HTML source stored on filesystem: Yes
- Published HTML stored on filesystem: Yes

### Filesystem paths
- Workspace: `backend/storage/sites/{site_id}/workspace/index.html`
- Published: `backend/storage/sites/{site_id}/published/index.html`

---

## POC Flow Audit

### 1. Create site
- Route/API: `POST /api/projects`
- Frontend: `frontend/src/views/DashboardView.vue`
- Backend: `handleCreateProject`
- Storage: creates workspace `index.html`
- Status: OK

### 2. Edit content
- Route/API: `GET /api/projects/:id`, `GET /api/sites/:id/workspace/index.html`
- Frontend: `frontend/src/views/EditorView.vue`
- Backend: `handleGetProject`, `handleGetWorkspaceIndex`
- Storage: workspace HTML path
- Status: PARTIAL
- Issue: backend only loads `index.html`

### 3. Save HTML
- Route/API: `PUT /api/sites/:id/workspace/index.html`
- Frontend: `frontend/src/stores/editor.js`
- Backend: `handleSaveWorkspaceIndex`
- Storage: workspace HTML path
- Status: OK
- Issue: only `index.html` persists

### 4. Publish
- Route/API: `POST /api/sites/:id/publish`
- Frontend: editor publish action and modal
- Backend: `handlePublishSite`
- Storage: workspace -> published copy
- Status: OK
- Issue: no template/data render phase

### 5. Open public website
- Route/API: `GET /sites/:slug`
- Frontend: publish modal open action
- Backend: `handleServePublishedSite`
- Storage: published HTML path
- Status: OK
- Issue: static-only output

### Manual test
1. Run `python run_project.py`
2. Open `http://localhost:5137`
3. Login with `ava` / `Welcome123!`
4. Open a site
5. Click `Edit site`
6. Edit `index.html`
7. Click `Save`
8. Click `Publish`
9. Click `Open Website`

### Expected URLs
- Frontend: `http://localhost:5137`
- API: `http://localhost:8080/api`
- Delivery: `http://localhost:8083/sites/{slug}`

### Known blockers
- No real multi-file persistence
- No asset pipeline
- No template renderer
- Static-only delivery

---

## Template Data Embedding Readiness

### Goal examples
- `{{ site.name }}`
- `{% for product in products %}`
- `{% for post in posts %}`
- `{% for plan in subscription_plans %}`

### Current state
- Products, posts, plans, and related metadata now exist in DB and APIs
- Publish currently copies static HTML only
- Delivery serves static published HTML only
- There is no template parser, renderer, or context builder

### Where rendering should happen
- Publish time is the cleanest place for first implementation
- The renderer should transform workspace source into published static HTML

### What must be added before embedding
1. Stable database strategy
2. Clean render context builder
3. Safe template grammar
4. Escaping rules
5. Loop/conditional handling
6. Preview mode for template rendering

### Readiness verdict
- Not ready yet

---

## Prioritized Next Steps

## P0 - Must fix before moving forward

### 1. Align database/runtime reality
- Why: PostgreSQL product direction conflicts with SQLite runtime
- Files/modules: `backend/internal/app/config.go`, `backend/internal/app/server.go`, `backend/internal/app/store.go`, `.env`, `README.md`
- Complexity: large
- Recommended Codex prompt topic: move backend from SQLite bootstrap to real PostgreSQL with tracked migrations

### 2. Standardize API contracts
- Why: frontend wiring will remain brittle otherwise
- Files/modules: `backend/internal/app/server.go`, frontend stores
- Complexity: medium
- Recommended Codex prompt topic: normalize API list/item/delete/error responses across all site-scoped domains

### 3. Make editor persistence truthful
- Why: current UI/backing model mismatch is a product trust issue
- Files/modules: editor view/store, workspace handlers
- Complexity: large
- Recommended Codex prompt topic: implement real multi-file workspace persistence and asset storage

## P1 - Needed for template/data embedding

### 4. Add publish render pipeline
- Why: embedding needs a render phase between source and published output
- Files/modules: publish flow, new renderer package
- Complexity: large
- Recommended Codex prompt topic: add publish-time render step for safe site data templates

### 5. Define safe template language
- Why: loops and interpolation need explicit guardrails
- Files/modules: future template package
- Complexity: large
- Recommended Codex prompt topic: design safe HTML-first template tag parser for site/product/post/plan data

### 6. Build render context from site-scoped data
- Why: renderer needs stable structured inputs
- Files/modules: backend domain queries
- Complexity: medium
- Recommended Codex prompt topic: add site-scoped render context builder for products, posts, and subscription plans

## P2 - Needed for better UX

### 7. Complete secondary CRUD flows
- Why: many admin tabs are still partial
- Files/modules: site dashboard, stores, backend admin handlers
- Complexity: medium
- Recommended Codex prompt topic: complete real CRUD and settings flows for all commerce/blog/subscription tabs

### 8. Upgrade editor engine
- Why: textarea limits future builder quality
- Files/modules: editor UI and state
- Complexity: medium
- Recommended Codex prompt topic: replace textarea with Monaco while preserving current save/publish flow

### 9. Tighten auth and ownership checks
- Why: site-scoped admin APIs need stronger boundaries
- Files/modules: backend auth and routing
- Complexity: medium
- Recommended Codex prompt topic: enforce site ownership and auth middleware on all admin APIs

## P3 - Later features

### 10. Add template snippet browser
- Why: authors need guided tag insertion
- Files/modules: editor UI
- Complexity: medium
- Recommended Codex prompt topic: add template snippet panel for site fields and data loops

### 11. Add rendered preview mode
- Why: authors need confidence before publish
- Files/modules: preview pipeline, renderer
- Complexity: large
- Recommended Codex prompt topic: render template tags in preview using sample/live site data

### 12. Add proper migration tracking and ops docs
- Why: schema growth now needs deployment discipline
- Files/modules: backend migrations, docs, scripts
- Complexity: medium
- Recommended Codex prompt topic: add schema_migrations and repeatable migration runner
