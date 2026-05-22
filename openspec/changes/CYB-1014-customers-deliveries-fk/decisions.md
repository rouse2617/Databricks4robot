# Decisions — CYB-1014

## 2026-05-22 — API contract files deferred (process gap)

- **Context**: Backend handlers and migration shipped; OpenAPI / api-guide / SDK not updated in the same PR.
- **Decision**: Track follow-up issue/PR for contract sync; repo rule updated in `docs/agents/AI-RULES.md` § API contract sync so future CYB work cannot repeat.
- **Rationale**: Handler-only merge left SDK and external consumers without a single source of truth.

## 2026-05-22 — Touch `backend/migrations/` (off-limits exception)

- **Context**: CYB-1014 requires DDL + backfill; same pattern as CYB-1013.
- **Decision**: Add `029_customers.sql` only.

## 2026-05-22 — Legacy customer_id format on backfill

- **Context**: Design slug regex `^[a-z][a-z0-9_-]{2,31}$` may not match all historical `deliveries.customer_id` values.
- **Decision**: Backfill inserts rows using existing `customer_id` strings as-is (no CHECK on table). API `POST /customers` enforces slug regex for new customers.
