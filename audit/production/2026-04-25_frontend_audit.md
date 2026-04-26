# Frontend Audit

Date: 2026-04-25

## Summary
The frontend is stronger than the backend ergonomically, but still depends on unstable backend contracts and carries broad views/stores that are not yet production-hardened.

## Status Matrix
- Route stability: PARTIAL
- Dashboard quality: PARTIAL
- Editor stability: PARTIAL
- File explorer truthfulness: PARTIAL
- Publish modal/public URL: PARTIAL
- Ecommerce/blog/subscription admin modules: PARTIAL
- Responsive layout: PARTIAL
- Blank page risk: PARTIAL
- API response handling: PARTIAL
- Stale state/cache handling: PARTIAL

## Findings

### 1. Vue architecture is workable
- Route-level lazy loading exists in `frontend/src/router.js`.
- Monaco editor is installed and used through the editor component stack.
- The editor experience is visually more production-like than the original POC.

### 2. Route protection is client-side only
- Router checks localStorage token and redirects.
- This is not security; it is UI gating.

### 3. Editor store is broad and request-heavy
- `frontend/src/stores/editor.js` loads the workspace file list, then fetches file content one file at a time.
- This is acceptable for a small workspace, but it will not age well with large workspaces or high latency.

### 4. Preview flow is improved but still simplified
- The store merges CSS/JS/HTML into `srcdoc` for preview.
- Published site opening uses the backend `public_url`.
- This is fine for development, but preview is still not a full release-faithful renderer.

### 5. Site dashboard is broad and partially shallow
- `frontend/src/views/SiteDashboardView.vue` is large and includes analytics, ecommerce, subscriptions, blogs, and overview sections.
- Some surfaces are strong enough for demoing, but the backend discipline under them is still uneven.

### 6. Error handling is not standardized enough
- Frontend frequently depends on permissive fallback logic (`data?.item || data || {}`).
- This reflects backend contract instability.

## Verdict
- Frontend production readiness: 6/10
- Main risk: frontend polish exceeds backend trustworthiness
