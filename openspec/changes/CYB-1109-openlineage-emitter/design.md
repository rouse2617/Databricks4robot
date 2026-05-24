# Design — CYB-1109

## Architecture Context
- **Constraints**: `asset_events` is the durable event source. `backend/internal/outbox/` is off-limits unless explicitly approved.
- **Goals**: Add a small OpenLineage emitter that can be enabled by config and tested without external services.
- **Non-Goals**: Replace DataBrew's PostgreSQL fact source, create a lineage UI, or require Marquez in dev.

## Affected Modules
- `backend/internal/openlineage/` — payload builder and HTTP emitter.
- `backend/internal/config/config.go` — disabled-by-default config flags.
- `backend/cmd/server/optional.go` — optional background wiring.
- `docs/review/api-guide.md` or backend README — operational notes for config.

## Architecture Decisions

### Decision 1: Use a thin internal OpenLineage package
- **Approach**: Build the minimal OpenLineage run event JSON internally and post with `net/http`.
- **Alternative**: Add a third-party OpenLineage Go SDK.
- **Rationale**: The design docs already note the Go SDK mismatch for this asset-centric model; a thin mapper avoids a new dependency.
- **Trade-off**: We own the small subset of schema mapping we emit.
- **Rollback**: Disable config or remove the optional wiring; no data changes.

### Decision 2: Disabled by default
- **Approach**: Gate the emitter with `OPENLINEAGE_EMITTER_ENABLED=true` plus an endpoint URL.
- **Alternative**: Always construct the emitter when outbox is enabled.
- **Rationale**: Marquez is not required in every environment and failures must not break normal search/indexing paths.

## Data Flow

```text
asset_events / outbox transport
  -> optional OpenLineage emitter
  -> map event + related run/asset identifiers
  -> POST OpenLineage JSON to configured endpoint
```

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| External endpoint down | Repeated delivery failures | Disabled by default, timeout HTTP calls, log failures |
| Payload lacks enough lineage context | Incomplete Marquez graph | Skip unsupported events and document mapping scope |
| Duplicates from at-least-once delivery | Duplicate events in consumer | Use stable run/event identifiers in payload facets |
