# CYB-3715 — Flatten mcap-file top-level columns into assets for direct filter/facet

## Why

Several mcap-file top-level columns are populated on every ingest but are **not filterable** through `/queries/run` because `assets` doesn't mirror them. Probing today returns **422 UNSUPPORTED_FIELD**:

- `camera_model` (e.g. "CyberCap2")
- `device_id` / `collector_id` / `scene_id` (uuid — for filter, not facet)
- `data_source` ("vendor" / "cybercap" / …)
- `collection_method` ("human" / "robot" / …)
- `source_platform` (currently inside `metadata.source_platform` — promote to first-class column so every producer standardizes on it)

These are the fields a data scientist uses to slice the corpus ("all CyberCap2 human-collected mcap from device X since 2026-07"). Today they're only visible on the detail page, one asset at a time.

CYB-3714b (CYB-3797) makes `task`/`source`/`city` queryable via tags; this ticket makes the structured mcap-file columns queryable via first-class asset columns. Together with CYB-3716 (metadata full-text) they close the "everything is searchable" story.

## What Changes

### 1. Migration — `backend/migrations/<timestamp>_flatten_mcap_columns_to_assets.sql`

Add nullable columns to `assets`:

```sql
ALTER TABLE assets
  ADD COLUMN camera_model      text,
  ADD COLUMN device_id         uuid,
  ADD COLUMN collector_id      uuid,
  ADD COLUMN scene_id          uuid,
  ADD COLUMN data_source       text,
  ADD COLUMN collection_method text,
  ADD COLUMN source_platform   text;

-- Partial indexes on the ones we expect to filter (high cardinality, low null count)
CREATE INDEX idx_assets_camera_model    ON assets(camera_model)    WHERE is_deleted = FALSE AND camera_model IS NOT NULL;
CREATE INDEX idx_assets_device_id       ON assets(device_id)       WHERE is_deleted = FALSE AND device_id IS NOT NULL;
CREATE INDEX idx_assets_scene_id        ON assets(scene_id)        WHERE is_deleted = FALSE AND scene_id IS NOT NULL;
CREATE INDEX idx_assets_source_platform ON assets(source_platform) WHERE is_deleted = FALSE AND source_platform IS NOT NULL;
-- data_source / collection_method / collector_id: low cardinality or per-uuid rare, no index needed
```

Backfill from `mcap_files` in the same migration:

```sql
UPDATE assets a
   SET camera_model      = m.camera_model,
       device_id         = NULLIF(m.device_id, '')::uuid,
       collector_id      = NULLIF(m.collector_id, '')::uuid,
       scene_id          = NULLIF(m.scene_id, '')::uuid,
       data_source       = m.data_source,
       collection_method = m.collection_method,
       source_platform   = COALESCE(m.metadata ->> 'source_platform', '')
  FROM mcap_files m
 WHERE a.mcap_file_id = m.mcap_file_id
   AND a.is_deleted   = FALSE
   AND m.is_deleted   = FALSE;
```

### 2. Handler mirror — write path stays in sync

`backend/internal/handlers/mcap/handler.go` `createFileTx`: when auto-deriving the placeholder raw_mcap asset (currently CYB-1217 block), copy the 7 columns from `f` (McapFile) onto `placeholder` (Asset). Also lift `metadata.source_platform` to the top-level column.

Any future handler that mutates mcap-file columns must be audited to propagate to the asset row.

### 3. Subscriber — sync to ES

`backend/internal/searchindex/` builds ES docs from asset rows. Add the 7 new fields to the doc projection so they're indexed. After the migration lands, trigger a full reindex via the existing admin reindex endpoint.

### 4. Planner facet whitelist

`backend/internal/queryplan/planner.go` — extend `PGSupportedFacetFields` (whichever the current map/slice is called) to include:
- `camera_model`, `data_source`, `collection_method`, `source_platform` (facet + filter)
- `device_id`, `collector_id`, `scene_id` (filter only — too many buckets to facet)

### 5. Frontend facet chips

`Frontend/src/components/assets/AssetsFacetSidebar.tsx` (or similar) — add 4 new chips (`camera_model`, `data_source`, `collection_method`, `source_platform`). Uuid fields stay filter-only, no chip.

### 6. Contract sync

Per `AI-RULES.md#api-contract-sync-mandatory`:
- `api/openapi.yaml` — schema of the 7 new asset fields
- `docs/review/api-guide.md` — one section documenting the new facet fields with a curl example
- `scripts/api-guide-smoke.sh` — assertions that `{field: "camera_model", op: "eq", value: "CyberCap2"}` no longer 422s

## Off-limits acknowledgement

- **`backend/migrations/`** — one new migration file. Per `AI-RULES.md#off-limits-zones`: **PR requires ≥2 reviewers.**
- Not touching `middleware/auth*`, `outbox/`, `.env*`, `schemas/pg-phase0.sql`.

## Acceptance

- Migration applies cleanly (dev CI fresh-PG17 replay + atlas.sum verify)
- `POST /queries/run where: {pred: {field: "camera_model", op: "eq", value: "CyberCap2"}}` returns hits
- Facet aggregation on `camera_model` / `data_source` / `collection_method` / `source_platform` returns non-empty buckets
- All existing 800+ pangzi mcap and their raw_mcap assets have populated columns (backfilled by migration)
- New mcap POST auto-populates the asset columns (verified by dev smoke: create mcap → query by `camera_model`)
- Frontend facet sidebar shows the 4 new chips + filter chip works
- ES reindex reports all rows re-indexed with new fields

## Rollback

- Data columns are nullable + backfill is idempotent, so a `git revert` is safe as long as no downstream reads have been added yet.
- If backend already reads the new columns and the revert removes them, need `ALTER TABLE DROP COLUMN` — write that as an explicit rollback migration.
- CloudSQL PITR is enabled — worst-case restore is well-defined.

## Out of scope

- JSONB metadata fields other than `source_platform` (e.g. `vibecap_tasks`, `location`, `collection_session_id`, `collector_height`) — those go through CYB-3714 (tags) or CYB-3716 (fulltext).
- Geographic queries (lat / lng radius) — separate ticket if needed.
- Retroactive rename of existing UUID-shaped fields (`device_id` etc. already exist on mcap-files as text; migration coerces to uuid via `NULLIF(...)::uuid`).
