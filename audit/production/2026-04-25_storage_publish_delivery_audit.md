# Storage / Publish / Delivery Audit

Date: 2026-04-25

## Summary
This is one of the strongest areas of the project now. It is not finished, but it has moved from POC-grade to a credible foundation.

## Status Matrix
- Canonical storage root: PARTIAL
- Competing storage roots removed from runtime: PARTIAL
- Workspace source on filesystem only: OK
- Website source stored in DB: NO
- Published releases immutable: PARTIAL
- Publish atomicity: PARTIAL
- Failed publish preserves live site: PARTIAL
- Delivery serves active release only: PARTIAL
- MIME handling for static assets: OK
- Multi-page navigation support: OK
- No restart needed after publish: PARTIAL
- Concurrent save/publish safety: PARTIAL

## Findings

### 1. Canonical storage model is mostly correct now
- Runtime config resolves repo-root `storage` via `resolveStorageRoot()` in `backend/internal/app/config.go`.
- Canonical paths now include tenant and site ids.
- However, repo still contains legacy/stale `backend/storage/...` artifacts, which is operationally confusing.

### 2. Workspace content is correctly filesystem-only
- Workspace file content is read from filesystem.
- `workspace_files` stores metadata, not file bodies.

### 3. Publish model is materially better
- `handlePublishSite` writes a new immutable release directory under `releases/{release_id}`.
- `sites.active_release_id` is switched after successful copy and DB transaction.
- A per-site PostgreSQL advisory lock rejects concurrent publish overlap.

### 4. Delivery model is much improved
- `handleServePublishedSite` resolves the site and its `active_release_id`.
- Delivery serves from the active release path.
- Delivery no longer depends on a mutable workspace path.

### 5. MIME handling is explicit and good
- `contentTypeForPath` handles HTML/CSS/JS/JSON/PNG/JPG/SVG and a sensible fallback.

### 6. Remaining concerns
- Legacy `published/` folders still exist and are still used as fallback in delivery if no active release is resolved.
- Cache headers are simple and acceptable, but not yet part of a broader CDN/proxy story.
- There is still no asset upload/storage lifecycle beyond simple filesystem assumptions.

## Verdict
- Storage/publish/delivery readiness: 6.5/10
- This area is closest to staging-grade, but still needs cleanup and more failure-mode tests.
