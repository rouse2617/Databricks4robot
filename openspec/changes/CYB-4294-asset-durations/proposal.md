# CYB-4294: batch asset-duration lookup — API + UI

## Context

Recurring workflow: take a list of asset ids, look up `duration_ms` for each, then filter by threshold or compute stats (batch-planning, cost estimates via duration × RTF, "only videos under 10 min", distribution analysis). Today engineers spin up a temp GKE pod and run `psql` by hand; non-tech teammates can't self-serve.

Two id shapes exist and users routinely mix them in one paste:
- `asset_id` — 8-char alphanumeric primary key
- `grace_video_id` — uuid text column added by CYB-4011

## Change

### Backend

New endpoint `POST /api/v1/assets/durations`.

Request:
```
{
  "ids": ["uYN6qys6", "019f8319-0ef3-7da0-815c-30f23d21e7f1", ...],
  "id_type": "auto" | "asset_id" | "grace_video_id",   // optional UI hint, not used by server
  "max_duration_ms": 600000,   // optional
  "min_duration_ms": 0          // optional
}
```

Response:
```
{
  "items": [{"input_id", "asset_id", "grace_video_id", "duration_ms", "duration_sec", "formatted"}],
  "missing_ids": ["019f...not-in-db"],
  "stats": {"matched_count", "missing_count", "total_ms", "mean_ms", "min_ms", "max_ms", "p50_ms", "p90_ms"}
}
```

SQL: single query, `WHERE asset_id = ANY($1) OR grace_video_id = ANY($1)` — one round-trip, mixed inputs supported natively.

Batch cap 5000 ids/request → 400 with a clear message when exceeded.

Auth: reuses the existing `assets:read` scope; already required for `/assets/*`.

### Frontend

New page `资产管理 → 时长批量查询` (route `/assets/durations`):
- Textarea to paste ids (newline / comma / space separated)
- `id_type` radio: auto / asset_id / grace_video_id (informational)
- Optional min/max duration inputs
- On submit → call API → render:
  - Stats card (matched / missing / total / mean / p50 / p90)
  - Fixed-bucket distribution bar chart (`<1min / 1-10min / 10-30min / 30-60min / 60min+`)
  - Results table (`input_id / asset_id / grace_video_id / duration_ms / formatted`), sortable, CSV export
  - Missing ids in a copyable column

## Non-goals

- Trigger batch dispatch from this page (existing batch flow already covers)
- Configurable bucket boundaries (wait for a second real request)
- Server-side pagination (5000-id cap keeps result ≤ 5000 rows, table can virtualize client-side)
