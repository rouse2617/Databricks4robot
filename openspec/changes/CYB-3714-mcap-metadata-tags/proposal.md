# CYB-3714 — Register `source` / `city` tags + backfill 140 pangzi mcap

## Why

pangzi VibeCap ingest (2026-07-21 onward, 140 mcap) stores business-critical labels inside `mcap_files.metadata` JSONB:

```json
{
  "source_platform": "vibecap",
  "vibecap_tasks": ["备餐操作", "台面清洁"],
  "location": {"address": "合肥新民医院, ..., 合肥市, ..."}
}
```

None of it is facetable or filterable today (`/queries/run` returns 422 UNSUPPORTED_FIELD for `source_platform`, `vibecap_tasks`, etc.). Meanwhile the `tag_registry` (CYB-3246) already exists as a first-class multi-value label system with facet + chip UI support — the right layer for these fields is **tags**, not JSONB queries.

`task` is already declared in `backend/config/tag_registry.yaml` as `type: string`, but `source` and `city` are not. And no existing pangzi mcap has any of the three tags populated.

## What Changes

**1. YAML baseline — declare `source` and `city`**

Add to `backend/config/tag_registry.yaml` `tags:` map:

```yaml
  source:
    description: "数据采集平台 (vibecap / grace / cybercap / ...)"
    type: string
  city:
    description: "采集地点(城市)"
    type: string
    max_length: 32
```

`task` is already present — no change needed there.

Both are `type: string` open-vocabulary (no fixed enum) — producers send free values, faceting is over the observed value distribution.

**2. Backfill script — populate the 3 tags on 140 pangzi mcap**

New file `scripts/backfill_pangzi_metadata_tags_dev.sh` (idempotent, re-runnable):

- Lists mcap where `owner="pangzi-consumer"` via `GET /api/v1/mcap-files?owner=pangzi-consumer&page_size=500`.
- For each mcap, extracts from `metadata` JSONB:
  - `vibecap_tasks[]` → one `task:<value>` tag per array entry (multi-value semantics)
  - `source_platform` → single `source:<value>` tag
  - Parse city from `location.address` — assume format `"<street>, ..., <city>, <province>, <postcode>, <country>"` and take the segment that ends in `市` (Chinese city suffix); if none, skip city tag.
- Posts each tag via `POST /assets/{mcap_file_id}/tags` with body `{key, value, source_type:"system", source_name:"backfill_cyb_3714"}`.
- `POST /assets/:id/tags` is an Upsert — re-running the script is a no-op for already-tagged mcap.
- Prints per-mcap status line: `mcap=<id> tasks=<n> source=vibecap city=<name>`; summary at end.

**3. No backend code change here**

`POST /assets/:id/tags` handler already exists and accepts arbitrary `(key, value)` pairs. Producer-side integration (VibeCap upload flow calling tags API on mcap create) is a separate ticket owned by the VibeCap team — this ticket only closes the **databrew** side of the loop (declare tags, backfill history).

## Impact

**Runtime**
- After merge: YAML change reaches the server via `deploy-dev.yml`. `tag_registry.source` and `tag_registry.city` become valid tag keys. `GET /api/v1/tag-registry` will surface them.
- After backfill runs: the 140 pangzi mcap have their `task` / `source` / `city` tags materialized in `asset_tags`. `POST /queries/run` with `{pred: {field: "tags.task", op: "eq", value: "备餐操作"}}` returns hits (was 200 total=0).
- Facet counts on the assets page automatically pick up the new tags (existing facet aggregator groups by tag key).

**Off-limits**
- None touched. YAML config change + shell script — no `middleware/auth*`, `outbox/`, `migrations/`, `.env*`, `schemas/pg-phase0.sql`.

**API contract**
- No new endpoints. `POST /assets/:id/tags` request shape is unchanged.
- `docs/review/api-guide.md` gets one line noting the two new baseline tags.

## Acceptance

- After deploy + backfill:
  - `GET /api/v1/tag-registry` includes `source` (type=string) and `city` (type=string, max_length=32).
  - Every pangzi mcap has at least one `task:*` tag and one `source:vibecap` tag.
  - `POST /queries/run` with `where: {pred: {field: "tags.task", op: "eq", value: "备餐操作"}}` returns > 0 hits.
  - `POST /queries/run` with `where: {pred: {field: "tags.source", op: "eq", value: "vibecap"}}` returns 140+ hits.
  - Backfill script exits with `pangzi mcap: N tagged (0 failed)`.

## Out of scope

- Producer-side change (VibeCap upload flow calling `POST /tags`) — separate ticket, external team.
- Geographic queries on `location.latitude/longitude` — needs PostGIS or ES geo mapping, separate ticket.
- Full-text search over `location.address` — covered by CYB-3713 (keyword mode) if we later extend `_fulltext` coverage to the JSONB metadata.

## Rollback

- `git revert` the merge commit — removes YAML additions. Since `task` was already a baseline tag, its rows survive; `source` and `city` rows also survive (asset_tags is just row data, not schema-constrained by the registry).
- If the tags themselves need to be un-populated, run `DELETE /api/v1/assets/:id/tags/{key}?source_type=system` in a loop — but there is no practical reason to do this; the tags are additive.
