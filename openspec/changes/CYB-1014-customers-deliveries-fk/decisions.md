# Decisions — CYB-1014

## 2026-05-22 — API contract follow-up (feat/CYB-1014-api-contract)

- **Context**: Initial backend merge omitted OpenAPI / api-guide / SDK.
- **Decision**: Same CYB-1014 change dir; added paths/schemas in `api/openapi.yaml`, api-guide §2.8, smoke hooks, `specs/customer-management/spec.md`. **SDK deferred** per team choice (not required for this issue).
- **Rationale**: Align with `docs/agents/AI-RULES.md` § API contract sync minimum (OpenAPI + api-guide + smoke).

## 2026-05-22 — Touch `backend/migrations/` (off-limits exception)

- **Context**: CYB-1014 requires DDL + backfill; same pattern as CYB-1013.
- **Decision**: Add `029_customers.sql` only.

## 2026-05-22 — Legacy customer_id format on backfill

- **Context**: Design slug regex `^[a-z][a-z0-9_-]{2,31}$` may not match all historical `deliveries.customer_id` values.
- **Decision**: Backfill inserts rows using existing `customer_id` strings as-is (no CHECK on table). API `POST /customers` enforces slug regex for new customers.
