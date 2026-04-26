# Deploy

This folder contains a Linux staging deployment foundation for Ubuntu VPS deployments.

## Included

- `env/` example environment files for staging and production
- `nginx/websitebuilder.conf` reverse proxy and static frontend example
- `systemd/` service units for API and delivery
- `scripts/` helper scripts for build, migrate, deploy, backup, restore, and smoke tests

## Assumptions

- Ubuntu VPS
- PostgreSQL installed and reachable
- Nginx installed
- systemd supervision
- non-root application user such as `sitecraft`
- repository checked out to `/opt/sitecraft`

## Runtime Shape

- API/admin traffic on port `8080`
- Public delivery on port `8083`
- Frontend static build served by Nginx
- Storage root at `/var/lib/sitecraft/storage`
- API and delivery run as separate binaries and separate systemd services.

## Quick Ops

- build: `deploy/scripts/build_linux.sh`
- migrate: `deploy/scripts/migrate.sh`
- backup DB: `deploy/scripts/backup_db.sh`
- restore DB: `deploy/scripts/restore_db.sh <file.sql|file.sql.gz>`
- backup storage: `deploy/scripts/backup_storage.sh`
- restore storage: `deploy/scripts/restore_storage.sh <file.tar.gz>`
- smoke check: `deploy/scripts/smoke_test.sh`

## Deployment Notes

- Build both Go binaries from `backend/cmd/api` and `backend/cmd/delivery`
- Run DB migrations before switching services
- Public delivery serves active published releases only
- Workspace content remains private behind authenticated admin APIs

## Restore Notes

- Verify the target environment file before running restore commands.
- Restore the database first, then restore storage if the release filesystem must match DB metadata.
- Storage restore extracts to the parent of `STORAGE_ROOT`, so keep `STORAGE_ROOT` pointed at the intended live path.
- DB restores run with `ON_ERROR_STOP=1` so failures stop immediately instead of partially applying.

## Scaling Notes

- API/admin traffic and public delivery already run on separate ports for reverse-proxy splitting.
- Published releases are immutable under `storage/sites/{tenant_id}/{site_id}/releases/{release_id}`.
- For multi-node delivery later, keep storage synchronized at the release-folder level and invalidate edge caches only after `active_release_id` changes.
