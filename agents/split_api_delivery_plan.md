# API/Delivery split plan (minimal change)

## Current backend structure
- `backend/cmd/api/main.go` starts one process with two listeners:
  - `API_PORT` server uses `srv.Router()`
  - `DELIVERY_PORT` server uses `srv.Router()`
- `backend/internal/app/server.go` has a single `Router()` that currently registers:
  - health routes (`/health`, `/healthz`, `/ready`)
  - public delivery routes under `/sites/*`
  - admin/auth/api routes under `/api/*`
- publish flow remains in `handlePublishSite` and delivery read flow remains in `handleServePublishedSite`.

## Current routes

### Health routes
- `GET /health`
- `GET /healthz`
- `GET /ready`

### Delivery/public routes
- `GET /sites/{siteID}`
- `GET /sites/{siteID}/`
- `GET /sites/{siteID}/{slug}`
- `GET /sites/{siteID}/{slug}/*`
- `GET /sites/{slug}`
- `GET /sites/{slug}/*`

### Admin/API routes
- `POST /api/auth/register`
- `POST /api/auth/login`
- Authenticated:
  - `GET /api/me`
  - `GET|POST /api/seed`
  - `GET /api/projects`
  - `GET /api/projects/{id}`
  - `POST /api/projects`
  - `... all /api/sites/{id}/workspace/*, /publish, preview, ecommerce, blog, subscription endpoints`

## Files to modify
- `backend/internal/app/server.go`
  - extract route registration into:
    - `registerHealthRoutes(router)`
    - `registerDeliveryRoutes(router)`
    - `registerAdminRoutes(router)`
  - keep business handlers unchanged
  - keep publish and delivery internals unchanged
- `backend/cmd/api/main.go`
  - convert to single API server on `API_PORT`
  - attach only admin + health routes
- `backend/cmd/delivery/main.go` (new)
  - start server on `DELIVERY_PORT`
  - attach only `/sites/*` + health routes
- `deploy/scripts/build_linux.sh`
  - build both binaries:
    - `bin/sitecraft-api`
    - `bin/sitecraft-delivery`
- `deploy/systemd/websitebuilder-api.service`
  - keep API binary mapping explicit
- `deploy/systemd/websitebuilder-delivery.service`
  - change ExecStart to delivery binary
- `deploy/README.md`
  - remove now-stale single-process note

## Risk points
- Route isolation risk:
  - API must not expose `/sites/*`
  - Delivery must not expose `/api/*`
- Health route compatibility:
  - keep health endpoints available on both services to avoid probe regressions
- Publish/delivery correctness:
  - delivery must still serve only active release (`active_release_id`)
  - never serve workspace files (must remain private behind admin auth endpoints only)
- Deploy/runtime risk:
  - systemd delivery unit currently points to API binary; must be corrected
  - build script must produce both binaries expected by services
- Frontend behavior risk:
  - `/api` and `/sites` ingress paths in nginx must remain unchanged

## Verification plan
- Run `go test ./...`
- Validate:
  - login, dashboard, editor, save, publish
  - public site and static assets (CSS/JS)
  - no `/sites` on API service
  - no `/api` on delivery service
  - no auth leakage / no 404 regressions
