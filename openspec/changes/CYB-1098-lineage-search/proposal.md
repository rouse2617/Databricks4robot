# Proposal — CYB-1098

## Why
Compliance operators need a read-only endpoint to trace asset dependency lineage across the catalog without manually walking each asset relation. The repo already has an early `/audit/lineage-search` skeleton, but CYB-1098 is not complete until recursive behavior, validation, documentation, tests, and dev smoke verification are contracted.

## What Changes

### New Capabilities
- Trace upstream, downstream, or bidirectional asset lineage from a starting asset.
- Bound recursive lineage traversal by a validated maximum depth.
- Return explicit empty results when no related assets exist.
- Reject malformed lineage search parameters with clear `400 INVALID_ARGUMENT` responses.

### Modified Capabilities
- Existing audit/discovery route wiring becomes production-ready for `/api/v1/audit/lineage-search`.

## Impact
- **Affected code**: `backend/internal/handlers/audit`, `backend/routes`, `api/openapi.yaml`, `docs/review/api-guide.md`, `scripts/api-guide-smoke.sh`
- **New APIs**: `GET /api/v1/audit/lineage-search`
- **Dependencies**: None

## Scope
- **In scope**: backend handler hardening, recursive CTE over `asset_relations`, upstream/downstream/both traversal, depth limits, validation errors, OpenAPI/api-guide/smoke sync, focused tests, dev deploy verification.
- **Out of scope**: audit event search (`CYB-1097`), SSE event stream (`CYB-1099`), frontend UI, SDK client parity, schema migrations, auth middleware, outbox internals.

## Success Criteria
- [ ] Clients can request lineage for a starting asset in `upstream`, `downstream`, or `both` direction.
- [ ] Recursive traversal follows `asset_relations` and returns bounded results with depth metadata.
- [ ] Empty lineage returns `200` with an empty `nodes` array and `count: 0`.
- [ ] Invalid `asset_id`, `direction`, or `depth` inputs return `400 INVALID_ARGUMENT`.
- [ ] The endpoint is documented in OpenAPI and the API guide with success, empty, and error examples.
- [ ] Smoke coverage verifies at least one successful query and one validation error on dev.

## Goals (SLO)
- **Latency**: Bounded by capped recursion depth; no unbounded graph traversal.
- **Concurrency**: Read-only endpoint; concurrent requests must not mutate asset state.
- **Quality**: Handler tests cover traversal direction, depth limiting, empty results, and validation failures.
