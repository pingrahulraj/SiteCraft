# Deployment Audit

Date: 2026-04-25

## Summary
The repository is not deployment-ready for Linux staging or production today. Local development is scripted; production operations are not.

## Status Matrix
- Linux build script: MISSING
- Production env example: MISSING
- Staging env example: MISSING
- Nginx config: MISSING
- systemd units: MISSING
- Production migration command: PARTIAL
- Backup script: MISSING
- Restore script: MISSING
- Log directory handling: PARTIAL
- Deployment documentation: MISSING
- Rollback plan: MISSING

## Findings

### 1. `run_project.py` is a dev runner, not a production deployment tool
- It is helpful for local bootstrap, init, test, and dev startup.
- It is tightly coupled to local development assumptions and Windows-friendly invocation patterns.
- It writes `GOCACHE` into `storage/.gocache`, which is acceptable for dev but not a deployment strategy.

### 2. Linux staging path is incomplete
- No systemd service units exist.
- No reverse-proxy config exists.
- No artifact packaging flow exists.
- No documented zero-downtime deploy procedure exists.

### 3. Process supervision is absent
- `backend/cmd/api/main.go` starts API and delivery listeners in one process.
- There is no graceful shutdown handling shown.
- There is no process manager config for restart policy or log rotation.

### 4. Logging is filesystem-local only
- Logs go into `storage/logs` or related local log files.
- There is no centralized logging guidance, rotation policy, or retention plan.

### 5. Backups and restore are not implemented
- No script or documentation covers PostgreSQL backups.
- No script or documentation covers filesystem backup of workspace/releases/assets.
- No restore validation flow exists.

## Direct Answers
- Can this be deployed to Ubuntu VPS today?
  Not safely as a customer-facing platform.
- Can it survive restart?
  Probably for basic development use, but this is not documented or operationally hardened.
- Can services restart cleanly?
  Not confidently enough for production; graceful shutdown and service units are missing.
- Can deploy happen without corrupting sites?
  The release model helps, but there is no full deployment/rollback procedure yet.

## Verdict
- Deployment readiness: 2/10
- Staging ready: NO
