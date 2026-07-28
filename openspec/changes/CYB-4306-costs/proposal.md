# CYB-4306: batch asset cost/gpu-min lookup — API + preset

## Context

FinOps + data-science recurring need: given N asset ids (asset_id or grace_video_id, mixed input allowed), a time window, and a group_by axis, return per-asset cost + GPU-seconds + CPU-seconds + run count in that window. Enables:
- "which of these 500 assets burned the most GPU-min last week?"
- "attribute this $X spike to specific assets"
- "sort by cost/second to spot expensive short clips"

Blocked-until: none — CYB-4304 shipped the `BatchAssetLookup` shell, this ticket just adds a new preset + backend endpoint. Data all present:
- `pipeline_run_nodes.estimated_cost_usd` (Argo-refresh computed, leaf-pod-only convention)
- `pipeline_run_nodes.resources_duration` JSONB with `cpu` (core-seconds), `nvidia.com/gpu` (GPU-seconds)
- `pipeline_runs.asset_ids` array (short asset_ids only)

## Change

### Backend

New endpoint `POST /api/v1/assets/costs` under `assets:read` scope.

Request:
```json
{
  "ids":       ["uYN6qys6", "019eda...uuid"],
  "id_type":   "auto",
  "start_at":  "2026-07-21T00:00:00Z",
  "end_at":    "2026-07-28T00:00:00Z",
  "group_by":  "asset"          // "asset" (default) | "asset_algo"
}
```

- `ids`: 1..5000, mixed asset_id and grace_video_id. Server resolves grace_video_id → asset_id via a first `SELECT asset_id, grace_video_id FROM assets WHERE asset_id = ANY($1) OR grace_video_id = ANY($1) AND is_deleted = FALSE`. Input ids that don't resolve go into `missing_ids`.
- `start_at`, `end_at`: **required**. RFC3339. Rejected if `end < start` or window > 90 days (safety cap so a bad request can't lock the pipeline_runs scan).
- `group_by`: default `"asset"`. `"asset_algo"` groups by (asset_id, algo_key) — each asset appears as N rows, one per algo it ran under in the window.

Response:
```json
{
  "items": [
    {
      "input_id":        "019eda...uuid",       // whichever request-side id resolved
      "asset_id":        "uYN6qys6",
      "grace_video_id":  "019eda...uuid",
      "total_cost_usd":  1.234,
      "gpu_sec":         720,                    // Σ resources_duration->>'nvidia.com/gpu'
      "cpu_sec":         480,                    // Σ resources_duration->>'cpu'
      "gpu_min":         12.0,                   // gpu_sec / 60
      "cpu_min":         8.0,
      "run_count":       3,
      "by_algo":         null                    // set only when group_by=asset_algo
    }
  ],
  "missing_ids":      ["019eda...not-in-db"],
  "filtered_out_ids": [],                        // present but no runs in window
  "stats": {
    "matched_count":         42,
    "missing_count":         2,
    "filtered_out_count":    5,
    "total_cost_usd":        18.502,
    "mean_cost_usd":         0.44,
    "p50_cost_usd":          0.32,
    "p90_cost_usd":          1.10,
    "total_gpu_sec":         14400,
    "total_cpu_sec":         3600,
    "total_run_count":       126
  }
}
```

When `group_by="asset_algo"`, each item's `by_algo` is populated with `[{algo_key, cost_usd, gpu_sec, cpu_sec, run_count}]` and there is one row per (asset, algo_key). No summary row per asset in that mode — the caller aggregates client-side or asks group_by=asset for the summary.

### SQL sketch

```sql
-- resolve request ids → asset_id set
WITH resolved AS (
  SELECT asset_id, grace_video_id
  FROM assets
  WHERE (asset_id = ANY($1) OR grace_video_id = ANY($1))
    AND is_deleted = FALSE
),
-- expand run.asset_ids array and join to leaf-pod nodes
per_asset AS (
  SELECT
    aid AS asset_id,
    n.template_name AS algo_key,           -- for group_by=asset_algo
    COALESCE(n.estimated_cost_usd, 0) AS cost_usd,
    COALESCE(CAST(n.resources_duration->>'nvidia.com/gpu' AS DOUBLE PRECISION), 0) AS gpu_sec,
    COALESCE(CAST(n.resources_duration->>'cpu' AS DOUBLE PRECISION), 0) AS cpu_sec,
    pr.id AS run_id
  FROM pipeline_runs pr
  JOIN pipeline_run_nodes n ON n.run_id = pr.id
  CROSS JOIN LATERAL unnest(pr.asset_ids) AS aid
  WHERE aid IN (SELECT asset_id FROM resolved)
    AND pr.finished_at BETWEEN $2 AND $3
    AND (n.type = 'Pod' OR (n.type = '' AND n.pod_name <> ''))     -- leaf-pods only (matches existing convention)
    AND n.estimated_cost_usd IS NOT NULL
)
SELECT
  asset_id,
  [algo_key when group_by=asset_algo,]
  SUM(cost_usd)   AS total_cost_usd,
  SUM(gpu_sec)    AS gpu_sec,
  SUM(cpu_sec)    AS cpu_sec,
  COUNT(DISTINCT run_id) AS run_count
FROM per_asset
GROUP BY asset_id [, algo_key]
```

Leaf-pod filter is critical — the existing single-run cost query at `pipeline_repo.go:767` uses the same predicate to avoid double-counting rollup nodes.

### Frontend

New preset `pages/presets/costs.tsx` consumed by `BatchAssetLookup` at `/assets?view=costs`:

- **Extra filter row**: date range picker (start/end) with default `now - 7d` → `now`, plus a Segmented for group_by (`按 asset` / `按 asset × 算法`). All three are required.
- **Columns** (group_by=asset): `input_id | asset_id | grace_video_id | total_cost_usd | gpu_min | cpu_min | run_count`, defaultSort `total_cost_usd desc`.
- **Columns** (group_by=asset_algo): additionally an `algo_key` column between grace_video_id and cost.
- **Histogram**: 5 cost buckets — `<$0.10 / $0.10-$1 / $1-$10 / $10-$100 / $100+`, hard-coded in the preset.
- **extraStats**: 总花销 USD, 均值 USD, P50 USD, P90 USD, 总 GPU-min, 总 CPU-min, 总运行次数.
- **CSV filename**: `asset-costs-<startDate>-<endDate>-N.csv`.
- **Tab wire-in**: `AssetsWorkbenchPage` adds a third tab `成本查询` (icon `DollarOutlined`), URL `?view=costs`.

## Cross-check with existing cost path

The `pipeline_repo.go:767` single-run cost query is the canonical convention. This new endpoint mirrors that predicate exactly (`n.type = 'Pod'` OR blank-with-pod-name, `estimated_cost_usd IS NOT NULL`). Result: the sum of costs across all assets appearing in the batch equals the sum of per-run costs for the union of matching runs (modulo per-asset attribution when a run touches multiple assets — see decisions.md).

## Non-goals

- Cross-window comparison / delta (this endpoint is one window at a time; caller runs twice for a diff)
- Spot/standard breakdown (already blended into `estimated_cost_usd`)
- Node-level drill-down (single-run detail page already has that)
- Prometheus / GMP live GPU-min (this is billed cost, from Argo `resourcesDuration`; live utilization lives elsewhere)
