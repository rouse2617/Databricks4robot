# Proposal — CYB-1097

## Why
Operators need a cross-asset audit search endpoint to investigate who or which run changed assets across the catalog. The repo already has an early `/audit/search` route, but CYB-1097 is not complete until the behavior is contracted, tested, documented, and smoke-verified.

## What Changes

### New Capabilities
- Search across `asset_events` by actor, time range, event type, and run id.
- Return a stable newest-first page of audit events with keyset pagination.
- Reject malformed filters with clear `400 INVALID_ARGUMENT` responses.

### Modified Capabilities
- Existing audit/discovery route wiring becomes production-ready for `/api/v1/audit/search`.

## Impact
- **Affected code**: `backend/internal/handlers/audit`, `backend/routes`, `api/openapi.yaml`, `docs/review/api-guide.md`, `scripts/api-guide-smoke.sh`
- **New APIs**: `GET /api/v1/audit/search`
- **Dependencies**: None

## Scope
- **In scope**: backend handler hardening, route safety, OpenAPI/api-guide/smoke sync, focused unit tests, dev deploy verification.
- **Out of scope**: audit lineage search (`CYB-1098`), SSE event stream (`CYB-1099`), frontend UI, SDK client parity, schema migrations, outbox internals.

## Success Criteria
- [ ] Clients can search audit events across assets by `actor`, `event_type`, `run_id`, `time_from`, and `time_to`.
- [ ] Responses are newest-first and include `items`, `limit`, and optional `next_cursor`.
- [ ] Invalid `limit`, `cursor`, or RFC3339 time filters return `400 INVALID_ARGUMENT`.
- [ ] The endpoint is documented in OpenAPI and the API guide with success and error examples.
- [ ] Smoke coverage verifies at least one successful query and one validation error on dev.

## Goals (SLO)
- **Latency**: Bounded by a capped page size; no unbounded response size.
- **Concurrency**: Read-only endpoint; concurrent requests must not mutate audit state.
- **Quality**: Handler tests cover filter parsing, pagination metadata, and validation failures.
