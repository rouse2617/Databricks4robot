# Decisions — CYB-1014

## 2026-05-22 — Touch `backend/migrations/` (off-limits exception)

- **Context**: CYB-1014 requires DDL + backfill; same pattern as CYB-1013.
- **Decision**: Add `029_customers.sql` only.

## 2026-05-22 — Legacy customer_id format on backfill

- **Context**: Design slug regex `^[a-z][a-z0-9_-]{2,31}$` may not match all historical `deliveries.customer_id` values.
- **Decision**: Backfill inserts rows using existing `customer_id` strings as-is (no CHECK on table). API `POST /customers` enforces slug regex for new customers.
