# Proposal — CYB-1109

## Why
DataBrew has asset, relation, and algorithm run facts in PostgreSQL, but no standard lineage event output for Marquez or other OpenLineage consumers.

## What Changes

### New Capabilities
- Emit OpenLineage-compatible events to a configured Marquez/OpenLineage HTTP endpoint.
- Map DataBrew `asset_events` and `algo_runs` context into OpenLineage job, run, input, output, and dataset fields where available.

### Modified Capabilities
- Optional outbox consumers gain a disabled-by-default lineage emitter alongside existing search subscribers.

## Impact
- **Affected code**: `backend/internal/openlineage/`, `backend/cmd/server/optional.go`, `backend/internal/config/config.go`, backend tests
- **New APIs**: None
- **Dependencies**: None planned; use standard `net/http`

## Scope
- **In scope**: disabled-by-default emitter, OpenLineage payload builder, HTTP delivery with timeout, config flags, unit tests.
- **Out of scope**: deploying Marquez, new database tables, frontend lineage UI, schema migrations, and producer payload changes.

## Success Criteria
- [ ] When disabled, the backend behavior is unchanged.
- [ ] When enabled and configured, supported asset events are converted to OpenLineage JSON and POSTed to the endpoint.
- [ ] Unsupported or incomplete events are skipped without failing unrelated outbox consumers.
- [ ] Unit tests cover payload mapping and HTTP success/failure handling.

## Goals (SLO)
- **Latency**: Emitter processing should add no blocking latency to HTTP writes.
- **Concurrency**: The emitter should run as an optional background consumer.
- **Quality**: Payload builder tests should be deterministic and not require a real Marquez service.
