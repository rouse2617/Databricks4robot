# Asset cost lookup spec — CYB-4306

## Endpoint

`POST /api/v1/assets/costs` (scope: `assets:read`)

### Request validation

- `ids` non-empty, ≤5000 → else `400 IdListRequired` or `400 IdListTooLarge`
- `start_at`, `end_at` — both required RFC3339 → else `400 InvalidTimeRange`
- `end_at >= start_at` → else `400 InvalidTimeRange`
- `end_at - start_at ≤ 90 days` → else `400 WindowTooLarge`
- `group_by ∈ {"", "asset", "asset_algo"}` (empty defaults to `"asset"`) → else `400 InvalidGroupBy`

### Server pipeline

1. **Resolve inputs to asset_ids** — single SELECT on assets where asset_id or grace_video_id matches. Build `map[input_id]resolved{asset_id, grace_video_id}`. Any input that finds no row goes into `missing_ids` and is dropped from step 2.

2. **Cost aggregate** — single SELECT joining `pipeline_runs` → `pipeline_run_nodes` with the leaf-pod predicate:
   - `n.type = 'Pod' OR (n.type = '' AND n.pod_name <> '')` (matches `pipeline_repo.go:767` convention exactly)
   - `n.estimated_cost_usd IS NOT NULL`
   - `pr.finished_at BETWEEN $start AND $end`
   - `CROSS JOIN LATERAL unnest(pr.asset_ids) AS aid` — flatten the array so we can group by aid
   - `WHERE aid = ANY($resolved_asset_ids)`

3. **Aggregation depends on group_by**:
   - `asset`: `GROUP BY aid` → SUM(cost), SUM(gpu_sec), SUM(cpu_sec), COUNT(DISTINCT run_id). One row per matched asset.
   - `asset_algo`: `GROUP BY aid, n.template_name` → same aggregates. Multiple rows per asset when it ran under multiple algos.

4. **Build items** — walk the request id list in insertion order:
   - Resolved and has aggregate → item with cost/gpu/cpu/run_count filled
   - Resolved but no aggregate → in `filtered_out_ids` (asset exists but had no runs in the window)
   - Unresolved → in `missing_ids`

5. **Stats** — over `items` (matched only):
   - `matched_count`, `missing_count`, `filtered_out_count`
   - `total_cost_usd`, `mean_cost_usd`, `p50_cost_usd`, `p90_cost_usd` — percentile_cont linear interpolation, sorted asc
   - `total_gpu_sec`, `total_cpu_sec`, `total_run_count`

### Response shape

Follows proposal.md exactly. Notable:
- `input_id` — whichever request-side id first resolved to the underlying asset. If the caller sends both an asset_id and its grace_video_id in the same request, one wins (whichever appears first in the request `ids`), the other is dropped from `missing_ids` since it resolved to the same asset.
- `gpu_min = gpu_sec / 60` and `cpu_min = cpu_sec / 60` computed server-side so the client doesn't format-shift.
- `by_algo` present only when `group_by=asset_algo`; otherwise `null` (not `[]`) so callers can `if item.by_algo` to branch.
- Percentile fields = 0 when items empty (NOT null / NaN).

### Cross-request behavior

- Duplicate input ids dedupe on the server before step 1 (keeps input_id in the response as the first-seen occurrence)
- When `group_by=asset` and an asset ran under 3 algos in the window: single item, single row per algo aggregated into `total_*`
- When `group_by=asset_algo`: 3 rows for the same asset_id, each with its own algo cost

## Per-asset attribution

When a pipeline run has `asset_ids = ['A', 'B', 'C']`, the cost is not split — each asset in the array is charged the full node cost. This over-attributes cost for multi-asset runs and is the same convention already used by the existing single-run cost path (`pipeline_repo.go:767` sums `estimated_cost_usd` per run without splitting across the asset_ids array). Decisions.md explains why we accept this.

## Frontend contract

The preset receives the response as-is and renders per the columns/histogram/csv config. UI-side behaviors:
- Date range picker default: `now - 7d` → `now` (both endpoints inclusive; UI trims to midnight/23:59:59 to match Argo `finished_at`)
- Empty state (matched=0): show a subdued alert `所选 ID 在该时间窗内无成本记录`; still render the missing_ids section
- CSV export dumps ALL items (both groupings; for asset_algo the algo_key column is included in the header)
