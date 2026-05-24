# Design — CYB-1099

## Architecture Context
- **Constraints**: `asset_events` is the source of truth for asset event history. `backend/internal/outbox/` is off-limits for this change.
- **Goals**: Make the existing SSE endpoint production-ready without adding new storage or background workers.
- **Non-Goals**: Fan-out infrastructure, browser UI subscription, or global event streams.

## Affected Modules
- `backend/internal/handlers/asset/handler.go` — validate inputs, check asset existence, stream poll results, and preserve cursor semantics.
- `api/openapi.yaml` — add the SSE operation and response/error documentation.
- `docs/review/api-guide.md` — add curl examples and operational notes.
- `scripts/api-guide-smoke.sh` — add lightweight stream smoke coverage.

## Architecture Decisions

### Decision 1: Poll `asset_events` inside the request stream
- **Approach**: Reuse the existing PostgreSQL-backed handler and poll bounded batches ordered by `event_seq ASC`.
- **Alternative**: Add a Pub/Sub-backed fan-out stream.
- **Rationale**: CYB-1099 only needs asset-scoped streaming and `asset_events` already stores the required resume cursor.
- **Trade-off**: Polling has a small visibility delay and ties long-lived connections to the backend process.
- **Rollback**: Remove the route and docs; no data rollback is needed.

### Decision 2: Treat `Last-Event-ID` as `event_seq`
- **Approach**: Parse `Last-Event-ID` as an integer and query for `event_seq > last`.
- **Alternative**: Use opaque event IDs.
- **Rationale**: `event_seq` is monotonic and already used by event pagination.
- **Risk**: Malformed values can otherwise create confusing resume behavior.
- **Rollback**: Return to starting from the oldest visible batch when no valid header is present.

## Data Flow

```text
Client
  -> GET /api/v1/assets/{id}/events/stream
  -> validate asset_id + optional Last-Event-ID
  -> confirm asset exists
  -> poll asset_events WHERE asset_id = $1 AND event_seq > cursor
  -> emit SSE frames and keepalive comments
```

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Long-lived streams hold resources | Backend connection pressure | Keep each poll bounded and use request cancellation |
| Missing asset streams forever | User sees silent no-op | Check asset existence before starting the stream |
| Invalid resume header silently ignored | Duplicate or confusing event replay | Return `400 INVALID_ARGUMENT` before streaming |
