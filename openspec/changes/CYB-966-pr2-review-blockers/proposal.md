# CYB-966 — PR #2 review blockers

## Problem

PR #2 review identified merge blockers:

1. Migration `025` drops `assets.status` while repository code still reads/writes the column.
2. Admin hard-delete routes accept `DATABREW_TOKEN` when `ADMIN_TOKEN` is unset in production.

## Scope

- Remove PostgreSQL `assets.status` usage; keep API `status` derived from `lifecycle_state`.
- Gate admin/internal routes in production on `ADMIN_TOKEN`.
- Renumber duplicate migrations; add CI fresh-DB job + migration filename guard.

## Out of scope

- Batch tag API endpoints (separate follow-up).
