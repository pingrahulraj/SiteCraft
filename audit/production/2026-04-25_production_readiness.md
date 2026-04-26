# Production Readiness Audit

Date: 2026-04-25
Project: `websitebuilder` / `SiteCraft`
Auditor scope: repository-only review, no code changes

## Executive Verdict
The project is not ready for real customer website migration yet. It has improved materially from the earlier POC state: PostgreSQL is now the active runtime database, workspace file persistence is filesystem-based, publish uses immutable release folders, and delivery serves an active release. That said, the platform is still a development-grade monolith with weak auth enforcement, incomplete tenant isolation, thin operational tooling, inconsistent docs, and no production deployment package.

## Scoring
- Architecture discipline: 5/10
- Multi-tenancy: 3/10
- Auth/security: 3/10
- Database/migrations: 6/10
- File storage safety: 7/10
- Publish safety: 7/10
- Delivery performance: 7/10
- Frontend stability: 6/10
- Testing coverage: 4/10
- Logging/auditability: 4/10
- Deployment readiness: 2/10
- Backup/restore readiness: 1/10
- Scalability readiness: 5/10

## Overall Scores
- Overall production readiness score: 4.6/10
- Staging readiness: NO
- Customer migration readiness: NO
- Paid customer readiness: NO

## What Is Production-Positive
- PostgreSQL is the active runtime DB in `backend/internal/app/config.go`.
- `schema_migrations` exists and migrations are recorded in `backend/internal/app/store.go`.
- Workspace source remains on filesystem, not in DB.
- Workspace file writes are atomic in `backend/internal/app/server.go`.
- Publish now creates immutable release folders and switches `active_release_id` after successful publish.
- Delivery serves the active release and not workspace content.
- MIME handling for HTML/CSS/JS/images is explicit.
- Basic smoke coverage exists for create/save/publish/delivery and a simple concurrent publish/read test.

## What Blocks Real Migration
- Admin APIs are effectively unauthenticated for most write paths. The router exposes site CRUD, workspace writes, publish, ecommerce, blog, and subscription APIs without auth middleware.
- Tenant isolation is not production-safe. Many handlers trust `site_id` alone and never verify the caller belongs to the tenant/site.
- The backend is still a single large server file with mixed routing, validation, SQL, storage, and response shaping.
- The repo still contains architecture drift and stale artifacts: README says SQLite, root `cmd/` and `internal/` residue remain, and backup server files exist in `backend/internal/app/`.
- Delivery and API are served from the same process binary and same router object. This is not fatal, but it is not a clean production split yet.
- There is no real deployment package for Linux: no Nginx config, no systemd units, no production env template, no deploy docs, no rollback doc, no backup/restore script.
- Secrets are development-grade: `.env` contains plaintext DB credentials and `JWT_SECRET=change-me`.
- Test coverage is still narrow: no real tenant isolation tests, no publish failure rollback test, no permission tests, no restore tests, no deployment tests.

## Final Verdict
- Can I deploy to staging now? NO
- Can I move internal websites now? NO
- Can I move customer websites now? NO
- Can I charge customers now? NO
- What is the next single best action?
  Lock down auth and tenant/site ownership enforcement across every admin API before any staging rollout.
