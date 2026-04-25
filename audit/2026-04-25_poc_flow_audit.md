# POC Flow Audit

## Core flow

### 1. Create site
- Route/API involved: `POST /api/projects`
- Frontend component involved: `frontend/src/views/DashboardView.vue`
- Backend handler involved: `backend/internal/app/server.go -> handleCreateProject`
- Storage path involved: creates workspace under `backend/storage/sites/{site_id}/workspace/index.html`
- Current status: OK
- Exact issue if broken: N/A

### 2. Edit content
- Route/API involved: `GET /api/projects/:id`, `GET /api/sites/:id/workspace/index.html`
- Frontend component involved: `frontend/src/views/EditorView.vue`
- Backend handler involved: `handleGetProject`, `handleGetWorkspaceIndex`
- Storage path involved: `backend/storage/sites/{site_id}/workspace/index.html`
- Current status: PARTIAL
- Exact issue if broken: editor UI implies multi-file workspace, but backend only loads `index.html`

### 3. Save HTML
- Route/API involved: `PUT /api/sites/:id/workspace/index.html`
- Frontend component involved: `frontend/src/stores/editor.js -> saveActive`
- Backend handler involved: `handleSaveWorkspaceIndex`
- Storage path involved: `backend/storage/sites/{site_id}/workspace/index.html`
- Current status: OK
- Exact issue if broken: only `index.html` is persisted; CSS/JS/files/assets are not persisted

### 4. Publish
- Route/API involved: `POST /api/sites/:id/publish`
- Frontend component involved: `frontend/src/stores/editor.js -> publish`, publish modal in `EditorView.vue`
- Backend handler involved: `handlePublishSite`
- Storage path involved: copies workspace `index.html` to `backend/storage/sites/{site_id}/published/index.html`
- Current status: OK
- Exact issue if broken: static-only publish; no template rendering or data embedding step

### 5. Open public website
- Route/API involved: `GET /sites/:slug`
- Frontend component involved: publish modal open/copy actions in `EditorView.vue`
- Backend handler involved: `handleServePublishedSite`
- Storage path involved: `backend/storage/sites/{site_id}/published/index.html`
- Current status: OK
- Exact issue if broken: serves static published HTML only

## Manual test
1. Start services:
   - `python run_project.py`
2. Open frontend:
   - `http://localhost:5137`
3. Login:
   - user `ava` or `ava@acme-studio.com`
   - password `Welcome123!`
4. Open a site dashboard and click `Edit site`
5. Change content in `index.html`
6. Click `Save`
7. Click `Publish`
8. Click `Open Website`

## Expected URLs
- Frontend: `http://localhost:5137`
- API: `http://localhost:8080/api`
- Delivery: `http://localhost:8083/sites/{slug}`

## Known blockers
- Editor only persists `index.html`
- Delivery is static-only and not ready for template tags
- Runtime launcher prints frontend URL `5137`, but `.env`/other values still mention other preview/publish ports and can confuse future work
