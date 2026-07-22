# Asset spec delta — CYB-3715b

## Asset schema (openapi.yaml `#Asset`)

New nullable string properties inserted between `task` and `version`:

| field | source | description |
|---|---|---|
| `camera_model` | `mcap_files.camera_model` | Mirrored on asset write. |
| `device_id` | `mcap_files.device_id` (uuid text) | Mirrored on asset write. |
| `collector_id` | `mcap_files.collector_id` (uuid text) | Mirrored on asset write. |
| `scene_id` | `mcap_files.scene_id` (uuid text) | Mirrored on asset write. |
| `data_source` | `mcap_files.data_source` | Mirrored on asset write. |
| `collection_method` | `mcap_files.collection_method` | Mirrored on asset write. |
| `source_platform` | `mcap_files.metadata.source_platform` | Lifted from JSONB on asset write. |

## Filter path (docs)

Top-level fields section in `api-guide.md` extended to list the 7 flatten fields alongside `lifecycle_state`, `asset_type`, `owner`, `duration_ms`, `created_at`, `updated_at`.

Notes:
- UUID columns (device_id / collector_id / scene_id) are filter-only; not exposed as facets (too high cardinality).
- Text columns (camera_model / data_source / collection_method / source_platform) are both filterable and facet-able.
- `mcap.<col>` remains a valid path for callers still going through the mcap_files subquery.

## Smoke regression

`scripts/api-guide-smoke.sh` gains a loop asserting HTTP 200 and non-zero filtered total for `camera_model` and `source_platform`, which are populated on dev (backfill: 1008 CyberCap2 + 797 vibecap).
