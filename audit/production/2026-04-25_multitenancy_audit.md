# Multitenancy Audit

Date: 2026-04-25

## Summary
The schema is tenant-aware, but request enforcement is not. Tenant isolation is not production-safe.

## Status Matrix
- `tenant_id` on tenant-owned tables: OK
- `site_id` on site-owned tables: OK
- File paths include `tenant_id + site_id`: OK
- Delivery prevents workspace exposure: PARTIAL
- Admin API ownership enforcement: BROKEN
- Cross-tenant access denial tests: MISSING

## Findings

### Schema coverage
- Core tenant/site tables consistently include `tenant_id` and `site_id` in `backend/internal/app/store.go`.
- Filesystem layout uses `storage/sites/{tenant_id}/{site_id}/...`, which is a strong foundation.

### API enforcement is weak
- `siteMeta(ctx, siteID)` in `backend/internal/app/server.go` resolves tenant and slug from site id only.
- Most handlers accept `site_id` from the URL and never verify the authenticated user belongs to that tenant/site.
- `handleListSites` selects by tenant slug query parameter, not authenticated principal.
- `handleGetSite`, workspace file CRUD, publish, products, posts, plans, and related admin endpoints rely on `site_id` and not caller ownership.

### Public delivery isolation
- Delivery resolves published site by `site_id + slug` or slug.
- Delivery serves from release/published paths and does not serve workspace directly.
- Path traversal checks exist in delivery and workspace path validation.
- This is better than earlier states, but public slug-only fallback still enlarges the surface.

## Direct Answers
- Can Tenant A access Tenant B data?
  Yes, if Tenant A can guess or obtain Tenant B site ids and call admin endpoints. There is no reliable server-side ownership gate.
- Can Tenant A publish Tenant B site?
  Yes in principle, for the same reason. Publish handler does not verify tenant/user authorization.
- Can public delivery leak workspace files?
  Not through intended delivery routes in the current code path; delivery reads published active release paths, not workspace.
- Is tenant isolation production-safe?
  No.

## Verdict
- Tenant isolation production-safe: NO
- Must-fix priority: P0
