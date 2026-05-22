# Proposal — CYB-1014

## Why

`deliveries.customer_id` is a bare TEXT with no referential integrity. Customers need a first-class table for SLA, compliance, and delivery filtering.

## What Changes

- `customers` table + backfill from distinct `deliveries.customer_id`
- FK `deliveries.customer_id` → `customers.customer_id`
- `POST/GET/PATCH /api/v1/customers`
- `GET /api/v1/deliveries?customer_id=...` filter
- Delivery commit validates customer exists

## Impact

- `backend/migrations/029_customers.sql`
- `backend/internal/models/`, `repository/`, `postgres/`, `handlers/customer/`, `handlers/delivery/`, `routes/`, `cmd/server/core.go`

## Out of scope

- `delivery_rules` table (CYB-1020)
- `contracts` table — `contract_id` stays TEXT on deliveries

## Success Criteria

See Linear CYB-1014 acceptance criteria.

## Context files

- `docs/review/unified-asset-catalog/design/customers-and-deliveries.md` (on `docs/CYB-983-unified-asset-catalog-prd` branch)
