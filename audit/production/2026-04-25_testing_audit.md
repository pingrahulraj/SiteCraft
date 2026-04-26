# Testing Audit

Date: 2026-04-25

## Summary
The project has real smoke tests now, but coverage is still narrow relative to production risk.

## Covered
- Health checks
- Database schema presence
- Storage presence
- Basic POC create/save/publish/delivery flow
- Workspace file CRUD basics
- CSS/JS MIME helper unit coverage
- Simple concurrent save/read/publish smoke coverage
- Frontend build command
- Go test suite execution

## Missing or Weak
- Tenant isolation / cross-tenant denial tests
- Admin authorization tests
- Publish failure rollback test
- Delivery explicitly proving workspace is never served
- Asset upload safety tests
- Release cleanup / retention tests
- Linux deployment smoke tests
- Service restart / graceful shutdown tests
- Backup / restore verification tests
- Frontend E2E tests for editor and dashboard flows

## Status Matrix
- Init/migrate/seed: PARTIAL
- Create site: OK
- Workspace file CRUD: PARTIAL
- Publish: PARTIAL
- Delivery: PARTIAL
- CSS/JS MIME: OK
- Multi-page navigation: PARTIAL
- Tenant isolation: MISSING
- Path traversal prevention: MISSING
- Failed publish safety: MISSING
- Concurrent save/publish: PARTIAL
- Frontend build: OK

## Key Risk
The current test set is enough to catch obvious regressions in the happy path, but not enough to trust the system with customer websites, tenant isolation, or operational recovery.

## Verdict
- Testing coverage: 4/10
- Production confidence from tests alone: NO
