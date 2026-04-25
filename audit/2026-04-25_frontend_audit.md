# Frontend Audit

## 1. Vue project structure
- PARTIAL
- Simple and workable, but large view files now carry too much product logic.

## 2. routing
- OK
- Core routes for login, dashboard, site dashboard, and editor are in place.

## 3. Pinia stores
- PARTIAL
- Separate stores exist for editor, ecommerce, blogs, and subscriptions, but contracts are inconsistent and some stores are broad aggregators.

## 4. API service layer
- PARTIAL
- Single axios client is fine for POC, but there is no domain service separation and no unified error mapping.

## 5. editor UX
- PARTIAL
- Visually improved and usable, but still based on textarea rather than Monaco and still not fully backed by real multi-file persistence.

## 6. file explorer UX
- PARTIAL
- Explorer interactions exist, but rename/delete/add/upload are mostly frontend-local.

## 7. save/validate/publish wiring
- PARTIAL
- Save/publish works for `index.html`. Validation is local-only and editor implies more filesystem support than backend provides.

## 8. publish success modal
- OK
- Modal exists and uses backend-provided URL.

## 9. ecommerce workspace
- PARTIAL
- Good UI shell. Some data now loads from backend, but parts of the workspace remain placeholder-driven.

## 10. blog workspace
- PARTIAL
- Good UI shell. Posts are real, but several secondary tabs still lack full editing workflows.

## 11. subscription stores workspace
- PARTIAL
- UI is polished, but wiring is thinner than Ecommerce/Blogs and relational workflows are still shallow.

## 12. UI density/polish
- OK
- Current density is compact and much closer to a sellable admin feel.

## 13. broken interactions
- PARTIAL
- Several buttons are placeholders or optimistic UI wrappers around incomplete backend depth.

## 14. mock vs real data
- PARTIAL
- Considerable improvement, but not fully purged. Some computed placeholders still fill gaps.

## 15. future template-snippet panel readiness
- MISSING
- No UI concept exists yet for inserting safe template tags or binding datasets.

## Area Status Summary
- Structure: PARTIAL
- Routing: OK
- Stores: PARTIAL
- API layer: PARTIAL
- Editor UX: PARTIAL
- File explorer: PARTIAL
- Save/validate/publish: PARTIAL
- Publish modal: OK
- Ecommerce: PARTIAL
- Blogs: PARTIAL
- Subscription Stores: PARTIAL
- Density/polish: OK
- Broken interactions: PARTIAL
- Mock vs real: PARTIAL
- Template-snippet readiness: MISSING
