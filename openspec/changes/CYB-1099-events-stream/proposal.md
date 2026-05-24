# Proposal — CYB-1099

## Why
Clients need a low-latency way to follow one asset's event stream without polling `/assets/{id}/events` repeatedly.

## What Changes

### New Capabilities
- Asset clients can subscribe to a server-sent events stream for one asset.
- Clients can reconnect with `Last-Event-ID` and resume after the last `event_seq` they received.

### Modified Capabilities
- The existing early SSE handler is hardened from skeleton behavior to a documented, tested, and smoke-covered API.

## Impact
- **Affected code**: `backend/internal/handlers/asset/handler.go`, `backend/routes/routes.go`, `api/openapi.yaml`, `docs/review/api-guide.md`, `scripts/api-guide-smoke.sh`
- **New APIs**: `GET /api/v1/assets/{id}/events/stream`
- **Dependencies**: None

## Scope
- **In scope**: asset-scoped SSE stream, `Last-Event-ID` resume, validation/error behavior, OpenAPI/api-guide/smoke coverage, focused backend tests.
- **Out of scope**: frontend consumption, SDK client wrapper, global cross-asset SSE streams, changing `asset_events` schema, touching `backend/internal/outbox/`.

## Success Criteria
- [ ] A valid asset stream returns `text/event-stream` frames with `id`, `event`, and JSON `data`.
- [ ] `Last-Event-ID` resumes from `event_seq > Last-Event-ID`.
- [ ] Invalid asset IDs or malformed `Last-Event-ID` fail with documented client errors before the stream starts.
- [ ] API contract files and smoke coverage document the stream behavior.

## Goals (SLO)
- **Latency**: New events should become visible within the handler polling interval.
- **Concurrency**: The implementation should avoid unbounded result sets per poll.
- **Quality**: Focused handler tests cover happy path, resume, and validation.
