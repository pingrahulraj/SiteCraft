# Architecture Audit

Date: 2026-04-25

## Summary
The codebase is still in an in-between state: better than a POC, but not yet a disciplined production platform. The publish/delivery model is moving in the right direction, but the service architecture, code organization, and API boundaries remain loose.

## Status Matrix
- Go project structure: PARTIAL
- API/editor/delivery separation: PARTIAL
- Monolith risk: BROKEN
- Frontend/backend contract stability: PARTIAL
- Windows dev to Linux production path: PARTIAL
- Support for many concurrent builders and visitors: PARTIAL

## Findings

### 1. Backend remains too monolithic
- `backend/internal/app/server.go` still owns routing, request parsing, validation, DB queries, storage operations, publish flow, delivery flow, logging helpers, and module CRUD.
- This is workable for development but risky for production debugging, incident response, and future change safety.

### 2. Editor and delivery responsibilities are conceptually separated, but not operationally separated enough
- Good: workspace traffic uses filesystem workspace paths and delivery uses active releases.
- Weak: `backend/cmd/api/main.go` starts both API and delivery from the same binary and same `srv.Router()`.
- Result: public traffic and admin traffic are still tightly coupled operationally.

### 3. Architecture drift is still visible
- Root-level `cmd/` and `internal/` residue remain beside active `backend/`.
- `README.md` still documents SQLite even though runtime now uses PostgreSQL.
- Backup/stale files remain in `backend/internal/app/` such as `server.go.5447608721189807935` and `server.go.5948513820665514175`.

### 4. Storage/publish architecture is significantly improved
- `backend/internal/app/site_files.go` now has workspace, published, and releases path helpers.
- `backend/internal/app/server.go` publishes to `releases/{release_id}` and delivery resolves `active_release_id`.
- This is a real step toward safe static hosting.

### 5. Frontend architecture is usable but broad
- Vue app uses route lazy loading and Pinia stores.
- The editor store in `frontend/src/stores/editor.js` is fairly broad and still does sequential file loading via many API requests.
- Site dashboard and blog/admin surfaces are large multipurpose views rather than well-isolated modules.

## Production Impact
- The current architecture can support continued development and controlled hardening.
- It is not yet clean enough for a reliable Linux staging or customer migration program.

## Verdict
- Architecture alignment: PARTIAL
- Production-grade service discipline: NO
