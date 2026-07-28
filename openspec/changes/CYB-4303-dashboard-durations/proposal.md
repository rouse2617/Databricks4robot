# CYB-4303: dashboard duration-distribution card

## Context

The current dashboard has two event-focused cards:
- "事件分布 (30 天)"
- "最新日事件类型分布 · <date>"

Both convey noise more than signal — event volumes fluctuate with pipeline runs, users can't map them to any decision. The user asked for a "数据时长分布" card instead: a global histogram of asset durations so both engineers and non-tech teammates can see at-a-glance how the corpus is shaped ("we have X hours of <1min clips, Y hours of 10-30min mcaps, …").

CYB-4294 already established the 5-bucket boundaries `<1min / 1-10min / 10-30min / 30-60min / 60min+`. This card reuses those same boundaries so single-asset lookups + fleet overview speak the same language.

## Change

### Backend

New endpoint under the existing dashboard read scope:

`GET /api/v1/dashboard/duration-distribution?asset_type=raw_mcap` (asset_type optional; default = all)

Response:
```
{
  "asset_type": "raw_mcap" | null,          // echoed for client display
  "buckets": [
    { "label": "<1min",     "lo_ms": 0,          "hi_ms": 60000,     "count": ..., "total_ms": ... },
    { "label": "1-10min",   "lo_ms": 60000,      "hi_ms": 600000,    "count": ..., "total_ms": ... },
    { "label": "10-30min",  "lo_ms": 600000,     "hi_ms": 1800000,   "count": ..., "total_ms": ... },
    { "label": "30-60min",  "lo_ms": 1800000,    "hi_ms": 3600000,   "count": ..., "total_ms": ... },
    { "label": "60min+",    "lo_ms": 3600000,    "hi_ms": null,      "count": ..., "total_ms": ... }
  ],
  "total_assets": ...,
  "total_ms": ...,
  "mean_ms": ...,
  "min_ms": ...,
  "max_ms": ...,
  "p50_ms": ...,
  "p90_ms": ...
}
```

One SQL query:
- `WHERE is_deleted = FALSE [AND asset_type = $1] AND duration_ms IS NOT NULL AND duration_ms >= 0`
- Bucket via CASE (mirrors the 5-bucket boundaries above)
- Aggregate: COUNT(*), SUM(duration_ms), MIN, MAX, `percentile_cont(0.5) WITHIN GROUP (ORDER BY duration_ms)`, `percentile_cont(0.9) WITHIN GROUP (ORDER BY duration_ms)`
- GROUP ROLLUP or use two subqueries — one bucket group by, one aggregate — union in Go if simpler

### Frontend

- Remove the two event cards (`EventDistributionCard` + `LatestDayEventTypeCard` or whatever their real names are — verify by reading DashboardPage first)
- Add `DurationDistributionCard` alongside the other dashboard cards
  - Reuses the same flexbox bar-chart style as CYB-4294's histogram (each bar labeled with count)
  - `Segmented` control at the top of the card to switch asset_type: `全部 | raw_mcap | segment | clip` — default 全部
  - Below the bar chart, a small footer row showing 总资产 / 总时长 / 均值 / P50 / P90 as `Statistic`s

### Non-goals

- Configurable bucket boundaries (still hard-coded)
- Time-series (this is a snapshot only)
- Filter by owner / tag (dashboard is fleet-scope, filtering belongs in the assets workbench)
