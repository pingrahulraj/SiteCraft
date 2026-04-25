# Backend Audit

## 1. Go project structure
- PARTIAL
- Active app lives under `backend/`, but root-level `cmd/` and `internal/` remain and increase confusion.

## 2. API route organization
- PARTIAL
- All routes are concentrated in one server file. Domain growth has outpaced maintainability.

## 3. service/handler/repository separation
- MISSING
- Handlers contain validation, SQL, defaults, and response shaping inline.

## 4. config loading
- PARTIAL
- Config loader exists, but it ignores most database env settings and still hardcodes SQLite behavior.

## 5. PostgreSQL connection
- BROKEN
- `.env` advertises PostgreSQL, but backend opens `modernc.org/sqlite` and uses `SQLITE_PATH`.

## 6. error handling
- PARTIAL
- Mixed styles: some endpoints use simple string errors, some use nested error objects. Consistency is low.

## 7. logging
- PARTIAL
- Minimal logging exists for publish flow and startup only. No structured request/domain logging.

## 8. middleware
- PARTIAL
- CORS exists. Auth middleware is not enforced consistently on admin APIs.

## 9. health endpoints
- OK
- `/healthz` exists.

## 10. publish API
- OK
- Reads workspace HTML, writes published HTML, verifies output, returns `public_url`.

## 11. delivery API
- OK
- Serves published `index.html` by slug from filesystem, static-only.

## 12. ecommerce APIs
- PARTIAL
- Broad route surface exists. Data model is ad hoc, some handlers are generic shortcuts, and response shape quality is uneven.

## 13. blog APIs
- PARTIAL
- Posts/categories/tags/authors/newsletter/members/analytics endpoints exist, but implementation is shallow and not cleanly separated.

## 14. subscription APIs
- PARTIAL
- Plans/subscribers/subscriptions/invoices/failed-payments/settings exist, but relationships and validations are light.

## 15. template/data embedding readiness
- BROKEN
- Publish and delivery are static-file flows. There is no render step, no template parser, no data binding stage, and no sandboxing model.

## 16. risks and fixes
- High risk: database direction mismatch (PostgreSQL planned, SQLite actual)
- High risk: one-file backend service with growing SQL complexity
- High risk: generic helper patterns now obscure entity-specific guarantees
- Medium risk: route surface is large but lacks auth/ownership rigor
- Medium risk: static-only delivery prevents future template injection

## Area Status Summary
- Structure: PARTIAL
- Routes: PARTIAL
- Separation: MISSING
- Config: PARTIAL
- PostgreSQL: BROKEN
- Error handling: PARTIAL
- Logging: PARTIAL
- Middleware: PARTIAL
- Health: OK
- Publish: OK
- Delivery: OK
- Ecommerce: PARTIAL
- Blogs: PARTIAL
- Subscriptions: PARTIAL
- Embedding readiness: BROKEN
