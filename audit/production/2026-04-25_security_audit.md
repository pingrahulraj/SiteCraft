# Security Audit

Date: 2026-04-25

## Summary
Security posture is still development-grade. The biggest issue is not cryptography or path traversal anymore; it is missing authorization.

## Status Matrix
- Authentication exists: PARTIAL
- Authorization enforcement: BROKEN
- Route protection: BROKEN
- Path traversal prevention: PARTIAL
- File extension allowlist: OK
- Upload safety: MISSING
- Secret handling: BROKEN
- CORS safety: PARTIAL
- Error leakage control: PARTIAL
- Public/private route separation: PARTIAL

## Findings

### 1. Auth exists but is barely enforced
- `handleLogin` issues JWT tokens.
- `handleMe` validates the bearer token.
- The router does not wrap admin routes in auth middleware.
- Result: most admin routes are effectively public to anyone who can reach the API.

### 2. No role/access model on write paths
- `tenant_users` exists in schema, but handlers do not check it.
- There is no owner/editor/viewer enforcement on site actions.

### 3. Secret handling is unsafe for production
- `.env` contains plaintext DB credentials.
- `.env` uses `JWT_SECRET=change-me`.
- There is no secret rotation or secure env handling guidance.

### 4. Path safety improved
- `allowedWorkspacePath` and `allowedWorkspaceDirectory` reject `..`, absolute paths, and invalid extensions.
- Delivery sanitizes request paths and blocks traversal attempts.
- This is one of the stronger parts of the current implementation.

### 5. Workspace is not publicly served in the current intended flow
- Delivery resolves release/published content, not workspace.
- This is good and should remain a hard rule.

### 6. Upload safety is not production-ready
- There is no real asset upload pipeline reviewable here.
- Asset metadata exists, but upload scanning/content checks are not present.

### 7. Template safety is not ready
- The project goal references future template rendering, but no safe production renderer is present.
- That is good in one sense because unsafe rendering is not yet live, but it means the future feature is still a security project.

## Explicit Answers
- Is workspace ever publicly served?
  Not in the current delivery path.
- Can delivery read arbitrary filesystem paths?
  Current traversal guards reduce this risk substantially, but the code should still be penetration-tested before production.
- Does a template renderer execute unsafe logic?
  No active renderer was found in the runtime path.

## Verdict
- Production security readiness: 3/10
- Single biggest blocker: missing authorization on admin APIs
