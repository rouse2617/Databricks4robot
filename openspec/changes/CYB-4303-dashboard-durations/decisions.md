# CYB-4303 decisions

## Why remove the event cards outright, not deprecate

They're not linked from anywhere else in the dashboard, they don't feed alerts, and no telemetry query shape depends on them. Keeping deprecated widgets around clutters the dashboard and confuses non-tech users. If someone actually needed event distribution as a card, we can add it back in ~30 min against the same events pipeline.

## Why match CYB-4294 bucket boundaries exactly

Two views of the same corpus should speak the same language: if the user sees "40 assets are 1-10min" on the dashboard, then picks 10 asset_ids and pastes them into 时长批量查询, they should see the same bucket labels on the histogram there. A single source of truth avoids "why is the dashboard bucket different from the lookup bucket" questions.

## Why hard-code buckets vs pull from CYB-4294 constants

Cross-module import (Backend Go handler importing a frontend constants file) is not viable. The buckets are hard-coded in the SQL CASE expression (backend) and mirrored in the response label strings. The frontend uses whatever the API returns and renders as-is — no client-side bucketization needed for the dashboard card (all 5 buckets ship in the response with zero counts if empty).

## Why percentile_cont over percentile_disc

`percentile_cont` returns interpolated values matching typical statistical convention. Postgres computes both efficiently on the same query. `_cont` is the same choice CYB-4294 makes (linear interpolation on sorted values), keeping the two features consistent.

## Why one endpoint per asset_type filter, not client-side split of "all"

The client can't compute per-asset_type stats from the "all" response — percentiles don't decompose across groups. A per-request `asset_type` filter is the natural granularity. Server load is fine (one indexed SQL per Segmented switch, few hundred ms max).

## Why not add an `owner` / `tag` filter

Dashboard is intentionally fleet-level. Adding owner/tag filters here would duplicate what the assets workbench (`/assets`) already does with its full facet sidebar. Users who want owner-scoped duration stats belong on the workbench; the dashboard exists to give a global snapshot.
