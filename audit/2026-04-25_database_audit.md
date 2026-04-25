# Database Audit

## 1. database bootstrap
- PARTIAL
- Bootstrap exists inside backend startup, but it is SQLite-only.

## 2. migrations
- PARTIAL
- Repeat-safe schema creation is embedded in code, but there is no explicit migration runner or migration history table.

## 3. schema_migrations
- MISSING
- No `schema_migrations` table or tracked migration versions.

## 4. sites table
- PARTIAL
- Equivalent `projects` table exists and drives site identity, status, slug, and preview URL.

## 5. ecommerce tables
- PARTIAL
- Tables now exist for products, orders, customers, coupons, store settings, shipping zones, and tax settings.
- Several constraints/indexes are present, but overall model quality remains POC-grade.

## 6. blog tables
- PARTIAL
- Tables exist for posts, categories, tags, authors, newsletter settings, members, and analytics.

## 7. subscription tables
- PARTIAL
- Tables exist for plans, subscribers, subscriptions, invoices, failed payments, and settings.

## 8. indexes
- PARTIAL
- Many `site_id` indexes exist. Coverage is improved but still hand-managed and not migration-tracked.

## 9. constraints
- PARTIAL
- Some uniqueness rules exist (`site_id + slug`, `site_id + code`, invoice uniqueness). Foreign keys exist in SQLite syntax. Business constraints remain mostly application-level.

## 10. seed/default data
- OK
- Seed data exists for tenants, users, projects, workspace files, products, posts, and newer admin datasets.

## 11. SQLite remnants
- BROKEN
- They are not remnants; SQLite is still the active database implementation.

## 12. HTML source storage check
- OK
- Website HTML source is not stored in the DB.

## 13. metadata-only rule
- PARTIAL
- Core website HTML follows filesystem storage. Blog post content is stored in DB, which is acceptable for blog metadata/content, but not for site HTML.

## 14. PostgreSQL readiness
- BROKEN
- `.env` is PostgreSQL-oriented, but the app does not open PostgreSQL, run PostgreSQL migrations, or use PostgreSQL types. JSONB intent is not implemented.

## Explicit confirmation
- Website HTML source stored in DB: No
- Website source stored on filesystem: Yes
- Published HTML stored on filesystem: Yes

## Notes
- Workspace HTML path: `backend/storage/sites/{site_id}/workspace/index.html`
- Published HTML path: `backend/storage/sites/{site_id}/published/index.html`
- Active DB path from config: `./storage/sitecraft.db`
