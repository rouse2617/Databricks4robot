# Proposal — CYB-1020

## Why

Deliveries can include assets that violate customer compliance (e.g. PII-tagged clips). Operators need declarative rules and a hard stop at commit time with readable violations.

## What Changes

### New Capabilities
- **delivery-rules**: `delivery_rules` table + CRUD API + PG evaluator at `POST /deliveries`
- **customer exclude_tags**: implicit block when asset tags match customer `exclude_tags`

### Modified Capabilities
- **deliveries**: commit rejects assets that match active `block` rules

## Impact

- `backend/migrations/032_delivery_rules.sql`
- `backend/internal/deliveryrules/`, `handlers/deliveryrule/`, `handlers/delivery/`
- `api/openapi.yaml`, `docs/review/api-guide.md`, `scripts/smoke-delivery-rules-dev.sh`

## Scope

- **In**: Table + POST/GET delivery-rules; block-mode check on `POST /deliveries`; structured `DELIVERY_RULE_FAILED` errors
- **Out**: C2 draft/commit multi-phase delivery; `warn`/`tag_only` enforcement; `POST /delivery_items` (no route yet); SDK (backend-only)

## Success Criteria

- [ ] Active rules + customer `exclude_tags` block delivery commit with asset + rule detail
- [ ] E2E: human tag `compliance.pii=true` + customer rule → 422 with violations
- [ ] OpenSpec + smoke on dev

## Context

- [`docs/review/unified-asset-catalog/design/customers-and-deliveries.md`](../../../docs/review/unified-asset-catalog/design/customers-and-deliveries.md) §3.4, §4.4, §5.1
