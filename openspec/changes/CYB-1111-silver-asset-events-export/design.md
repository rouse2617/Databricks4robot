# Design — CYB-1111

## Architecture Context
- **Constraints**: Bronze ingestion already writes `bronze_asset_events`; BigLake/Iceberg may be disabled in local/dev environments.
- **Goals**: Provide an explicit Silver asset-events export path and make it observable from lakehouse table metadata.
- **Non-Goals**: Running a production backfill automatically or adding new public query endpoints.

## Affected Modules
- `deploy/k8s/jobs/biglake-silver-gold-build-once.yaml` — existing one-shot Silver/Gold build path.
- `deploy/cloudrun/bronze-incremental/` — incremental lakehouse job context and deploy env notes.
- `backend/internal/handlers/lakehouse/handler.go` — table listing behavior.
- `docs/review/api-guide.md` and `scripts/api-guide-smoke.sh` — docs and verification.

## Architecture Decisions

### Decision 1: Keep Silver export as batch materialization
- **Approach**: Build Silver from Bronze with deterministic dedupe/current-state semantics.
- **Alternative**: Query Bronze directly for every analytical endpoint.
- **Rationale**: Bronze can contain expected duplicates after retry; Silver gives stable semantics for downstream analytics.
- **Trade-off**: Silver is eventually consistent with Bronze.
- **Rollback**: Disable or skip the Silver job; Bronze remains untouched.

### Decision 2: Use existing table listing response shape
- **Approach**: Extend lakehouse table visibility without adding a new endpoint.
- **Alternative**: Add a dedicated Silver status API.
- **Rationale**: `/lakehouse/tables` already answers operational table availability and row counts.

## Data Flow

```text
PostgreSQL asset_events
  -> Bronze incremental ingest
  -> bronze_asset_events
  -> Silver export job
  -> silver_asset_events_current
  -> lakehouse table visibility / downstream analytics
```

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Silver table absent in dev | Smoke false failures | Use warning/tolerant checks when lakehouse is disabled |
| Bronze duplicates | Incorrect analytics if queried directly | Deduplicate by stable event sequence/current-state semantics in Silver |
| Batch lag | Analytics trail realtime writes | Document batch cadence and use Bronze sync metrics for freshness |
