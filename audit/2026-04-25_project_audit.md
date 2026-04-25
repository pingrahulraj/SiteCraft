# Project Audit

## 1. Executive summary
The repository currently functions as an HTML-first website builder POC with working site creation, HTML editing, save, publish, and delivery. The project has materially drifted from the stated Go + Vue + PostgreSQL architecture and still runs on SQLite plus filesystem storage. Ecommerce, Blogs, and Subscription Stores now have visible admin surfaces and some persistence, but the implementation is inconsistent, partially mocked, and not yet ready for safe template/data embedding.

## 2. Current product state
- Core builder flow exists and is demonstrable.
- Admin feature surface is broad.
- Architecture is still POC-grade rather than production-ready.

## 3. What currently works
- Login and signup basic flow
- Site listing and site dashboard routing
- Site editor route and UI shell
- Save of `index.html` to filesystem
- Publish copies workspace HTML to published HTML
- Delivery serves published HTML by site slug
- Runtime launcher and log files
- Basic CRUD persistence for some products/posts/plans data

## 4. What is partially working
- Editor behaves like a file-based IDE in UI, but backend only persists `index.html`
- Ecommerce workspace mixes real API data with UI-only placeholders
- Blogs workspace mixes real API data with placeholder actions and incomplete CRUD wiring
- Subscription Stores have backend tables and APIs, but frontend remains partially wired and still assumes placeholder relationships
- Response formats are inconsistent across legacy and newer endpoints

## 5. What is broken
- PostgreSQL is not actually used despite `.env` and product direction
- Root README and env/config/runtime are contradictory
- Multiple root-level folders (`cmd/`, `internal/`) suggest architecture residue and repo drift
- Editor file explorer implies multi-file persistence that does not exist server-side
- Delivery remains static-only and cannot embed data into published HTML

## 6. What is overbuilt
- Dashboard surface area expanded faster than backend consistency
- Large number of CRUD endpoints relative to actual product-critical POC needs
- UI breadth for commerce/blog/subscriptions exceeds reliability of current services and schema discipline

## 7. What is missing
- Real PostgreSQL bootstrap, connection, migration tracking, and usage
- Repository/service separation in backend
- Unified API error/response contract
- Full multi-file workspace persistence
- Asset persistence and delivery
- Safe template rendering pipeline for `{{ }}` and loop tags
- Authorization and site ownership checks beyond basic token presence

## 8. Architecture alignment score out of 10
4/10

## 9. POC readiness score out of 10
7/10

## 10. Product risk score out of 10
8/10

## 11. Recommended immediate focus
Freeze new feature breadth and stabilize the foundation: backend data/storage consistency, response contracts, PostgreSQL migration path, and multi-file/editor truth model before attempting template/data embedding.
