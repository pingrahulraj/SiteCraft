# SiteCraft

SiteCraft is an HTML-first multi-tenant website builder with a Go backend, PostgreSQL runtime storage, and a Vue 3 frontend.

## Runtime Architecture

- `backend/cmd/api` - API and public delivery process entrypoint
- `backend/internal/app` - backend routes, auth, workspace, publish, and delivery logic
- `frontend` - Vue 3 admin UI
- `storage/sites/{tenant_id}/{site_id}/workspace` - editable site source files
- `storage/sites/{tenant_id}/{site_id}/releases/{release_id}` - immutable published releases
- Public delivery path: `/sites/{site_id}/{slug}`

## Data Model

- PostgreSQL stores tenants, users, memberships, site metadata, workspace metadata, and release metadata.
- Website HTML/CSS/JS source stays on the filesystem.
- Public delivery serves only the active published release, never the workspace.

## Local Development

Requirements:

- Go
- Node.js
- PostgreSQL
- Python 3

Run the local workspace:

```powershell
python run_project.py run-dev
```

Useful commands:

```powershell
python run_project.py init
python run_project.py test
python run_project.py reset-db
```

`reset-db` and demo seeding are development-only flows.

## Deployment Foundation

Linux staging examples now live under [`deploy/`](./deploy):

- `deploy/env/`
- `deploy/nginx/`
- `deploy/systemd/`
- `deploy/scripts/`

See [`deploy/README.md`](./deploy/README.md) for Ubuntu VPS staging guidance.
