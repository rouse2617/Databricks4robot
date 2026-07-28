# CYB-4306 decisions

## Why POST not GET

Same reason CYB-4294 chose POST: id lists can hit 5000 + a date range in a query string is url-shaped noise. POST body keeps ids out of access logs and works cleanly through any proxy.

## Why require `start_at` and `end_at` (no default window server-side)

The default window is the biggest source of billing-attribution disputes. Client sets defaults ("7 days back") explicitly so the request is self-describing in access logs. If the server picked "the last N days" every call would return different totals depending on the wall clock.

## Why the 90-day window cap

`pipeline_run_nodes` is the largest table on dev (millions of rows). A full-year scan by asset_id is ~5s and unattributed. Cap keeps the endpoint bounded; the FinOps use case is week/month grain anyway. A caller who wants a year makes 4 quarterly calls.

## Why no per-asset cost split for multi-asset runs

Pipeline runs sometimes fan across `asset_ids = ['A', 'B', 'C']` (e.g. one node processes 3 inputs). The Argo `resourcesDuration` and derived cost is per-node, not per-asset. Splitting cost/3 is arbitrary — the wallclock cost was one pod, not three. Options considered:
1. Full cost per asset (over-attributes) — CHOSEN. Matches the existing single-run `pipeline_repo.go:767` convention exactly.
2. Cost / len(asset_ids) — arbitrary and misleading (the pod's cost doesn't scale with input count)
3. Report both — doubles the response payload for a case that's rare

Response includes `run_count` per asset so callers can spot the over-attribution (an asset with runs shared across many assets shows an inflated cost/run ratio).

## Why leaf-pod predicate (`type = 'Pod' OR blank-with-pod-name`)

Argo DAG/Steps nodes carry a rollup `resourcesDuration` that IS the sum of children's — including them double-counts. The existing single-run cost aggregate (`pipeline_repo.go:767`) uses the exact same predicate. Diverging would produce inconsistent totals across the two features.

## Why `template_name` as the `algo_key`

Argo template name is the closest stable identifier — matches how the pipeline editor names steps. Not `pipeline_node_id` (per-run instance ids don't aggregate across runs) and not `image` (varies with build tags).

## Why the client-only histogram, not server-computed

Same as CYB-4294's durations: cost buckets are UX concerns, response payload already carries per-item costs, client can rebucket on filter/sort without a round-trip.

## Why NOT stream / paginate

At the 5000-id cap the response is ≤ 5000 rows × ~200 bytes = ~1 MB, single POST is fine. `by_algo` mode can push it higher if avg algos-per-asset is high, but the same 5000-id cap keeps it bounded.

## What we're NOT doing here

- Multi-window comparison. Two calls + client diff.
- GKE / GMP live utilization. This is billed Argo cost, not utilization.
- Cost budget alerts. That's an operator concern, not a lookup.
