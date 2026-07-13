# search Specification — CYB-3384 delta

## MODIFIED Requirements

### Requirement: Facet aggregation engine selection

The system SHALL choose the facet aggregation engine based on real-time PG↔ES sync health,so facet counts stay consistent with the authoritative list count:

- When `sync_health.pg_es_gap == 0` (PG and ES in sync),facet aggregation runs on Elasticsearch (fast path,default for production).
- When `sync_health.pg_es_gap > 0`,facet aggregation SHALL fall back to PostgreSQL `GROUP BY` on the assets table for supported fields (`asset_type`,`lifecycle_state`,`owner`,`env`);unsupported fields (`mcap.*`,`tag.*`) SHALL return an empty bucket list plus a `warnings` entry.
- Environment variable `DBK_FACET_ENGINE=auto|es|pg` SHALL override the automatic selection for diagnosis:`auto` uses sync health (default),`es` forces ES,`pg` forces PG.
- `debug_plan.steps` SHALL reflect the chosen engine (`{"engine":"postgres","mode":"facet"}` when PG facet is used).

### Requirement: Sync health cache

The system SHALL maintain an in-memory `SyncHealthCache` refreshed from `GET /api/v1/search/sync-progress` at 30-second cadence:

- Initial refresh happens synchronously at server startup so the first request doesn't skip the sync-health check.
- Refresh failures preserve the previous value and increment `sync_health_refresh_errors_total`.
- Planner reads the cache without contention (RWMutex + atomic snapshot).
