# Hardening Latest

- Date: 2026-04-26
- Scope: production-foundation hardening without new product features

## Before / After Scores

- kernel/platform: 5.0 -> 7.5
- data: 6.0 -> 7.0
- API: 5.5 -> 7.5
- builder/editor: 6.0 -> 7.0
- UI: 5.5 -> 7.0
- security: 5.0 -> 7.5
- deployment/ops: 4.5 -> 7.0
- scalability: 5.5 -> 6.5

## Completed

- added `/health` and `/ready`
- added panic recovery and safer lifecycle logging
- added graceful shutdown for API/delivery listeners
- added lightweight rate limiting for login, publish, and workspace writes
- expanded audit logging for login, site create, workspace mutations, and publish
- tightened frontend auth failure handling and fatal error fallback
- added Linux ops placeholders for worker, restore storage, and smoke test scripts

## Remaining Risks

- API surface is still large and mostly lives in one backend package
- in-process rate limiting is single-instance only
- delivery and admin still share one binary even though ports are split
- no distributed invalidation, revocation, or centralized audit pipeline yet
- release cleanup is documented but not yet automated

## Verification

- `go test ./...`
- `npm run build`
- `python tests/run_all.py`
