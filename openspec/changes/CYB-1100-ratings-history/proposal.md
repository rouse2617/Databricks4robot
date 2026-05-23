# Proposal — CYB-1100

## Why
Operators need a logical-asset-level quality trend endpoint so they can compare rating signals across revisions without opening each revision one by one.

## What Changes

### New Capabilities
- asset-management: `GET /api/v1/logical-assets/{id}/ratings-history` returns a read-only, oldest-to-newest rating history for a logical asset family.
- asset-management: each non-deleted revision is represented even when it has no `asset_metrics` `rating.*` rows yet, so clients can distinguish "no ratings" from "missing logical asset".

### Modified Capabilities
- Existing logical asset route wiring becomes contracted, documented, tested, and smoke-verified for ratings history.

## Impact
- **Affected code**: `backend/internal/handlers/asset`, `backend/routes`, `api/openapi.yaml`, `docs/review/api-guide.md`, `scripts/api-guide-smoke.sh`
- **New APIs**: `GET /api/v1/logical-assets/{id}/ratings-history`
- **Dependencies**: None

## Scope
- **In scope**: response contract, backend read behavior, OpenAPI/api-guide/smoke sync, focused handler/repository tests, dev verification.
- **Out of scope**: schema migrations, new rating tables, frontend UI, SDK client parity, lakehouse analytics, delivery-rule evaluation changes, CYB-1125, and any `.claude` worktree.

## Success Criteria
- [ ] Clients can fetch ratings history by `logical_asset_id` through `GET /api/v1/logical-assets/{id}/ratings-history`.
- [ ] The response is ordered by `revision ASC`, then rating metric/source/recorded time ASC for deterministic rendering.
- [ ] Revisions with no ratings are returned with an empty `ratings` array.
- [ ] Unknown or fully deleted logical assets return `404 ASSET_NOT_FOUND`; blank or malformed ids return `400 INVALID_ARGUMENT`.
- [ ] OpenAPI, API guide, smoke script, and OpenSpec behavior delta are updated in the same change.
- [ ] Tests cover no history, invalid id, missing asset, revisions with no ratings, and sort/order semantics.

## Goals (SLO)
- **Latency**: Bounded by one logical asset family; no cross-catalog scan.
- **Concurrency**: Read-only endpoint; concurrent rating writes may appear in a later request but must not mutate asset state.
- **Quality**: Contract tests ensure the JSON shape matches OpenAPI and distinguishes empty ratings from not found.
