# Dashboard duration-distribution spec delta — CYB-4338

Amends CYB-4303's `/api/v1/dashboard/duration-distribution` behavior. Only the
bucket set changes; the response envelope, per-bucket schema, permissive
`asset_type` handling, `hi_ms=null` top bucket, and zero-on-empty stats are all
unchanged.

## Changed behavior: 5 → 10 buckets

- WHEN the endpoint returns `buckets`
  THEN it MUST always contain exactly **10** rows, in this fixed ascending order,
  even when empty (`count=0, total_ms=0`):

  | # | label | lo_ms | hi_ms |
  |---|-------|-------|-------|
  | 0 | `<1m`    | 0        | 60000     |
  | 1 | `1-5m`   | 60000    | 300000    |
  | 2 | `5-10m`  | 300000   | 600000    |
  | 3 | `10-15m` | 600000   | 900000    |
  | 4 | `15-20m` | 900000   | 1200000   |
  | 5 | `20-25m` | 1200000  | 1500000   |
  | 6 | `25-30m` | 1500000  | 1800000   |
  | 7 | `30-45m` | 1800000  | 2700000   |
  | 8 | `45-60m` | 2700000  | 3600000   |
  | 9 | `60m+`   | 3600000  | null      |

- Bucket edges are half-open `[lo_ms, hi_ms)`; the top bucket (`60m+`) is
  open-ended (`hi_ms=null`).
- Bucket `label`/edges in `models.DurationBucketOrder` and the repo SQL `CASE`
  MUST stay in lockstep — a relabel lands on both sides at once (asserted by
  `repos_test.go`).

## Unchanged (regression guard)

- `total_assets / total_ms / mean_ms / min_ms / max_ms / p50_ms / p90_ms` —
  same fields, same zero-on-empty semantics. Percentiles remain
  `percentile_cont(0.5|0.9)` over raw `duration_ms` (NOT bucket-derived).
- `asset_type` echo + permissive filter (nonsense value → `total_assets=0`, never 400).
