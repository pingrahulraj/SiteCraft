# Database Audit

Date: 2026-04-25

## Summary
Database posture has improved significantly. PostgreSQL is active, migrations exist, and most domain tables are tenant/site scoped. The remaining issues are operational discipline and some PostgreSQL-specific implementation choices.

## Status Matrix
- PostgreSQL active runtime DB: OK
- SQLite not active runtime: OK
- `schema_migrations` exists: OK
- Repeat-safe migrations: PARTIAL
- Reset-db dev-only posture: PARTIAL
- Seed data idempotent: PARTIAL
- Index coverage for tenant/site/slug lookups: PARTIAL
- CockroachDB future compatibility: BROKEN

## Findings

### 1. PostgreSQL is active runtime
- `backend/internal/app/config.go` and `openPostgres()` clearly use `pgx`.
- This corrects the earlier POC mismatch.

### 2. Migrations are embedded and tracked
- `backend/internal/app/store.go` defines migration steps and records them in `schema_migrations`.
- Migrations are idempotent enough for development iteration.

### 3. Seed path is development-oriented
- Seed inserts a default tenant, user, site, module data, and a published copy.
- Good for demo bootstrap, but not appropriate as-is for production startup assumptions.

### 4. Index coverage is improving but still selective
- There are indexes on many `site_id` paths and newer indexes for `active_release_id` and `status, slug, id`.
- There is still no broad evidence of query review or EXPLAIN-driven tuning.

### 5. PostgreSQL-specific choices block easy CockroachDB portability
- Advisory locks in `backend/internal/app/server.go` are PostgreSQL-specific.
- JSONB and some SQL patterns are also PostgreSQL-specific.
- This is fine if PostgreSQL is the committed platform, but it means “Cockroach later” is not a free future path.

## Explicit Answers
- Website HTML source stored in DB: No
- Website HTML source stored on filesystem: Yes
- Published HTML stored on filesystem: Yes

## Verdict
- Database readiness: 6/10
- Good enough for staging hardening work, not enough for customer migration
