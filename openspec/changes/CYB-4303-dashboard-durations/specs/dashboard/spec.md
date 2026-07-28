# Dashboard duration-distribution spec — CYB-4303

## New endpoint

`GET /api/v1/dashboard/duration-distribution?asset_type=<optional>`

- Scope: reuse whatever the existing `/dashboard/*` endpoints require (do NOT invent a new scope).
- `asset_type` — optional query param. Whitespace trimmed. When empty/absent, aggregate across all `is_deleted=FALSE` assets. When non-empty, filter with `WHERE asset_type = $1`. Any string is accepted (server-side does not enumerate types — a nonsense value simply returns `total_assets=0`, mirroring how existing dashboard endpoints handle absent data).
- Response shape (spec exactly — do not add/remove fields):

```json
{
  "asset_type": "raw_mcap",
  "buckets": [
    { "label": "<1min",    "lo_ms": 0,       "hi_ms": 60000,   "count": 12,   "total_ms": 250000 },
    { "label": "1-10min",  "lo_ms": 60000,   "hi_ms": 600000,  "count": 34,   "total_ms": 5100000 },
    { "label": "10-30min", "lo_ms": 600000,  "hi_ms": 1800000, "count": 56,   "total_ms": 44000000 },
    { "label": "30-60min", "lo_ms": 1800000, "hi_ms": 3600000, "count": 21,   "total_ms": 55000000 },
    { "label": "60min+",   "lo_ms": 3600000, "hi_ms": null,    "count": 3,    "total_ms": 15000000 }
  ],
  "total_assets": 126,
  "total_ms": 119350000,
  "mean_ms": 947222,
  "min_ms": 500,
  "max_ms": 6500000,
  "p50_ms": 850000,
  "p90_ms": 2900000
}
```

- Buckets are always all 5, in order, even when empty (`count=0, total_ms=0`).
- `hi_ms: null` for the top bucket (`60min+`) — clients render it as an open-ended upper bound.
- When `total_assets=0`, all percentile/mean/min/max fields return 0 (do NOT return NaN or null).
- `asset_type` field in response echoes the query, or is `null` when absent — clients use this for display consistency.

## SQL sketch

```sql
WITH filtered AS (
  SELECT duration_ms
  FROM assets
  WHERE is_deleted = FALSE
    AND duration_ms IS NOT NULL
    AND duration_ms >= 0
    AND ($1 = '' OR asset_type = $1)
),
bucket_agg AS (
  SELECT
    CASE
      WHEN duration_ms < 60000        THEN '<1min'
      WHEN duration_ms < 600000       THEN '1-10min'
      WHEN duration_ms < 1800000      THEN '10-30min'
      WHEN duration_ms < 3600000      THEN '30-60min'
      ELSE                                '60min+'
    END AS label,
    COUNT(*)              AS count,
    COALESCE(SUM(duration_ms), 0) AS total_ms
  FROM filtered
  GROUP BY label
),
overall AS (
  SELECT
    COUNT(*) AS total_assets,
    COALESCE(SUM(duration_ms), 0) AS total_ms,
    COALESCE(AVG(duration_ms), 0)::bigint AS mean_ms,
    COALESCE(MIN(duration_ms), 0) AS min_ms,
    COALESCE(MAX(duration_ms), 0) AS max_ms,
    COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY duration_ms), 0)::bigint AS p50_ms,
    COALESCE(percentile_cont(0.9) WITHIN GROUP (ORDER BY duration_ms), 0)::bigint AS p90_ms
  FROM filtered
)
SELECT
  (SELECT total_assets FROM overall),
  (SELECT total_ms FROM overall),
  (SELECT mean_ms FROM overall),
  (SELECT min_ms FROM overall),
  (SELECT max_ms FROM overall),
  (SELECT p50_ms FROM overall),
  (SELECT p90_ms FROM overall);
-- + separate query for buckets (with LEFT JOIN over a fixed 5-bucket table
--   OR merge missing buckets client-side in the repo layer to guarantee 5 rows)
```

Repo layer fills missing buckets with zeros before returning so callers always see all 5 rows in order.

## Frontend card

- Card title: `数据时长分布`
- `Segmented` at top-right: `全部 (default) / raw_mcap / segment / clip`. Changing it triggers a fresh fetch.
- Bar chart: 5 flexbox bars, same style as CYB-4294 durations page. Each bar labeled `<label>\n<count>` (or count as tooltip when narrow). Y-axis implicit (bar height = count / max_count).
- Below the chart, a `Row gutter=[16,8]` of `Statistic`s: 总资产, 总时长, 均值, P50, P90. Durations displayed as `formatDurationMs` (reuse the durations preset's helper if extracted, otherwise inline in a card-local helper).
- Empty state: when `total_assets=0`, render `<Empty description="所选类型下无资产" />` in place of the chart; still show 0-valued stats below (or hide the stats row entirely — either is fine, pick whichever looks less noisy).

## Card placement

Replace the two event cards. If the existing grid uses `Col span=12` for those, keep the same span for the new card so surrounding layout stays unchanged. If the two cards occupied `Col span=12` each side-by-side, the new card takes the full `Col span=24` row (wider makes the histogram readable).
