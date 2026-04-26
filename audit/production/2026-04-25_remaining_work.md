# Remaining Work

Date: 2026-04-25

## P0 - must fix before Linux staging

### 1. Enforce auth and tenant/site ownership on all admin APIs
- Why it matters: current API exposure is the largest customer-blocking risk.
- Affected files/modules: `backend/internal/app/server.go`, auth middleware layer, tenant/user lookup helpers.
- Risk if ignored: cross-tenant reads/writes, unauthorized publish, customer data compromise.
- Complexity: large
- Recommended Codex prompt topic: add auth middleware and tenant/site ownership checks across all admin routes

### 2. Clean repository and runtime architecture drift
- Why it matters: stale files, incorrect docs, and old storage artifacts raise deployment risk.
- Affected files/modules: `README.md`, root `cmd/`, root `internal/`, stale `server.go.*`, storage docs.
- Risk if ignored: operator error, wrong deployment assumptions, accidental use of outdated code paths.
- Complexity: medium
- Recommended Codex prompt topic: clean architecture drift and align docs with PostgreSQL plus release-based delivery

### 3. Create a real Linux deployment package
- Why it matters: no reliable path to Ubuntu staging exists today.
- Affected files/modules: new ops docs, Nginx config, systemd units, env examples, deploy scripts.
- Risk if ignored: staging rollout instability and unrepeatable manual deployment.
- Complexity: medium
- Recommended Codex prompt topic: add Ubuntu deployment package with systemd, Nginx, env examples, and rollout docs

### 4. Replace startup seed assumptions with explicit dev-only bootstrap controls
- Why it matters: production processes should not auto-seed demo state.
- Affected files/modules: `backend/internal/app/store.go`, `run_project.py`, environment gating.
- Risk if ignored: accidental demo data behavior in non-dev environments.
- Complexity: medium
- Recommended Codex prompt topic: make seed/bootstrap explicitly dev-only and safe for staging production environments

## P1 - must fix before moving first real website

### 5. Add publish rollback/failure-path tests
- Why it matters: customer trust depends on failed publishes not affecting live sites.
- Affected files/modules: `backend/internal/app/*_test.go`, smoke tests.
- Risk if ignored: hidden release corruption or false confidence in atomic publish.
- Complexity: medium
- Recommended Codex prompt topic: add publish failure rollback and active-release integrity tests

### 6. Add tenant isolation and authorization tests
- Why it matters: security controls are unproven until tested.
- Affected files/modules: backend integration tests, smoke tests.
- Risk if ignored: regressions could silently re-open cross-tenant access.
- Complexity: medium
- Recommended Codex prompt topic: add cross-tenant denial and auth enforcement tests for admin APIs

### 7. Separate delivery service operationally from admin API
- Why it matters: public traffic should not compete with write-heavy editor traffic.
- Affected files/modules: `backend/cmd/api/main.go`, new delivery entrypoint, deploy config.
- Risk if ignored: public latency and operational coupling under load.
- Complexity: medium
- Recommended Codex prompt topic: split admin API and public delivery into separate binaries/services

### 8. Standardize API contracts
- Why it matters: frontend currently compensates for inconsistent response shapes.
- Affected files/modules: `backend/internal/app/server.go`, frontend stores.
- Risk if ignored: brittle UI behavior and slower future hardening.
- Complexity: medium
- Recommended Codex prompt topic: normalize API response and error contracts across site-scoped endpoints

## P2 - must fix before paid customers

### 9. Add backup and restore tooling for DB plus filesystem storage
- Why it matters: customer websites require disaster recovery.
- Affected files/modules: new ops scripts/docs, storage and PostgreSQL backup procedures.
- Risk if ignored: irreversible data loss.
- Complexity: medium
- Recommended Codex prompt topic: add backup restore scripts and recovery docs for PostgreSQL and site storage

### 10. Add production logging, rotation, and retention guidance
- Why it matters: incident response requires structured operational observability.
- Affected files/modules: logging helpers, deploy docs, logrotate/systemd config.
- Risk if ignored: hard-to-debug failures and log growth issues.
- Complexity: small
- Recommended Codex prompt topic: harden structured logging and production log management for API and delivery services

### 11. Harden asset pipeline and upload safety
- Why it matters: public asset handling becomes a security and abuse vector.
- Affected files/modules: asset endpoints, storage rules, validation layer.
- Risk if ignored: malicious uploads, broken sites, storage abuse.
- Complexity: medium
- Recommended Codex prompt topic: add safe asset upload validation and delivery hardening

### 12. Add frontend end-to-end regression tests
- Why it matters: editor/save/publish/public URL flows are customer-critical.
- Affected files/modules: new E2E test suite, CI scripts.
- Risk if ignored: broken customer flows slipping past build-only checks.
- Complexity: medium
- Recommended Codex prompt topic: add Playwright E2E coverage for editor save publish and dashboard workflows

## P3 - later improvements

### 13. Refactor backend into handlers/services/repositories
- Why it matters: maintainability and onboarding will degrade otherwise.
- Affected files/modules: `backend/internal/app/server.go` and related extracted packages.
- Risk if ignored: rising regression rate and slower delivery.
- Complexity: large
- Recommended Codex prompt topic: split monolithic server.go into domain handlers services and repositories

### 14. Introduce release retention and cleanup policy
- Why it matters: immutable releases will grow storage over time.
- Affected files/modules: publish/release management, ops docs.
- Risk if ignored: storage bloat and manual cleanup burden.
- Complexity: medium
- Recommended Codex prompt topic: add release retention cleanup and rollback-safe pruning

### 15. Prepare safe template rendering architecture
- Why it matters: products/blogs/subscriptions rendering is part of the future product direction.
- Affected files/modules: future renderer package, publish pipeline.
- Risk if ignored: later feature work will stall or arrive insecurely.
- Complexity: large
- Recommended Codex prompt topic: design safe publish-time template rendering architecture with strict escaping and context rules
