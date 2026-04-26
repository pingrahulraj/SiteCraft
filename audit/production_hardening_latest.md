# Production Hardening Latest

- Date: 2026-04-26
- Scope: P0 staging hardening only

## Fixed

- Added JWT auth middleware for admin APIs
- Added tenant and site ownership enforcement for site-scoped admin routes
- Restricted development seed behavior to development mode
- Added Linux staging deployment foundation under `deploy/`
- Updated README to reflect PostgreSQL, filesystem workspace, release delivery, and public route format

## Tests Added Or Updated

- smoke auth coverage
- smoke tenant isolation coverage
- smoke authenticated create/save/publish flow
- smoke authenticated workspace file CRUD flow
- smoke concurrency flow updated for authenticated admin access

## Tests Run

- `go test ./...`
- `npm run build`
- `python tests/run_all.py` against a restarted local API with `RUN_LIVE_HTTP_TESTS=true`

## Remaining Risks

- Backend route organization is still monolithic
- Registration flow still assumes tenant slug onboarding model
- No production-grade rate limiting or session revocation yet
- Smoke tests still depend on a live local stack for full HTTP coverage

## Staging Readiness

- Improved materially from prior audit
- Suitable for staging preparation after verification passes
- Suitable for Linux staging packaging work, not for paid-customer production yet
