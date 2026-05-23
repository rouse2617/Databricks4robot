# Design — CYB-1097

## Architecture Context
- **Constraints**: `asset_events` is the existing append-only event/audit stream for asset lifecycle changes. `backend/migrations/` and `backend/internal/outbox/` are off-limits unless explicitly approved.
- **Goals**: Expose a read-only audit search API using existing indexed columns and a bounded page size.
- **Non-Goals**: Add new tables, change event production semantics, implement lineage search, or build a frontend.

## Affected Modules
- `backend/internal/handlers/audit/handler.go` — validate query filters, query `asset_events`, and shape the response.
- `backend/routes/routes.go` — ensure audit routes are registered safely.
- `api/openapi.yaml` — publish the endpoint contract and response schema.
- `docs/review/api-guide.md` — add curl examples and validation notes.
- `scripts/api-guide-smoke.sh` — add dev smoke checks for success and validation error.

## Architecture Decisions

### Decision 1: Source audit search from `asset_events`
- **Approach**: Query `asset_events` directly for cross-asset audit rows and expose row-level actor/run/time dimensions.
- **Alternative**: Query `audit_events`, which stores coarse handler-level audit calls.
- **Rationale**: CYB-1097 asks for actor, time range, event type, and run id dimensions; those dimensions already exist on `asset_events`, and adjacent asset event APIs use the same stream.
- **Trade-off**: Results represent catalog event history, not every coarse handler audit record.
- **Rollback**: Remove the `/api/v1/audit/search` route and OpenAPI/api-guide entries; no data rollback is needed.

### Decision 2: Use keyset pagination by `event_seq`
- **Approach**: Return events ordered by `event_seq DESC`; `cursor` means fetch rows with `event_seq < cursor`.
- **Alternative**: Use page/offset pagination.
- **Rationale**: `asset_events` is append-only and can grow large; keyset pagination avoids offset drift and scales with the existing monotonic sequence.
- **Trade-off**: Clients cannot jump directly to page N.

### Decision 3: Keep page size capped
- **Approach**: Default `limit` to 50 and cap at 200.
- **Alternative**: Allow arbitrary limits.
- **Rationale**: The endpoint is cross-asset; a cap prevents accidental large scans and oversized responses.

## Data Flow

```text
Client -> GET /api/v1/audit/search?filters
  -> Static token auth
  -> audit handler validates filters
  -> PostgreSQL asset_events query ordered by event_seq DESC
  -> JSON response with items + limit + optional next_cursor
```

## Data Model Changes
- None.

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Unfiltered searches can scan many recent events | Slow dev/prod requests | Keep response bounded and use keyset pagination; prefer indexed filters in docs |
| Actor values may be empty for older events | Some filters return fewer rows than expected | Document that `actor` filters `actor_id` when present |
| Existing handler uses direct SQL | Harder to mock than a repository interface | Add focused handler tests around validation and query behavior; avoid introducing a broad abstraction unless implementation demands it |
