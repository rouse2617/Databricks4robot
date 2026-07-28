# Asset spec delta — CYB-4294

## New endpoint

`POST /api/v1/assets/durations` (requires `assets:read`)

### Request

- `ids` — required, non-empty, ≤5000, string array. May contain any mix of `asset_id` (8-char) and `grace_video_id` (uuid text).
- `id_type` — optional. `"auto"` (default) | `"asset_id"` | `"grace_video_id"`. Server does not filter by this; it is preserved as a UI hint. WHERE clause is always `asset_id = ANY($1) OR grace_video_id = ANY($1)`.
- `max_duration_ms` — optional int64 ≥0. Rows with `duration_ms > max_duration_ms` are excluded from `items` (they still count as "not in the requested slice", but every unmatched request id — whether missing or filtered out — appears in `missing_ids` if the caller wants strict accounting; the initial pass puts filtered-out matches into a distinct `filtered_out_ids` list).
- `min_duration_ms` — optional int64 ≥0. Same semantics, other end.

### Response

- `items` — one row per matched-and-in-range asset. Fields:
  - `input_id` — the id from the request that resolved to this row (matches by whichever column hit). If the same asset was named by both its asset_id and grace_video_id in the request, one arbitrary input wins; the other collapses.
  - `asset_id`, `grace_video_id`, `duration_ms`
  - `duration_sec` — `duration_ms / 1000`
  - `formatted` — human string, `Xm Ys` for <60min, `Xh Ym` for ≥60min, `0s` when duration_ms=0
- `missing_ids` — ids from the request that did NOT match any row (deleted / typo / never existed).
- `filtered_out_ids` — ids that DID match a row but the row's `duration_ms` fell outside `[min_duration_ms, max_duration_ms]`.
- `stats` — computed over `items` only (i.e. after filtering).
  - `matched_count` — `len(items)`
  - `missing_count` — `len(missing_ids)`
  - `filtered_out_count` — `len(filtered_out_ids)`
  - `total_ms`, `mean_ms`, `min_ms`, `max_ms`, `p50_ms`, `p90_ms` — 0 when items empty
  - Percentiles: linear interpolation, sorted asc.

### Errors

- `400 IdListRequired` — empty `ids`
- `400 IdListTooLarge` — `len(ids) > 5000`
- `400 InvalidDurationRange` — either bound negative, or `min > max` when both set
- `500` — repo error

### Contract note

Dedup: server dedupes `ids` before the SQL round-trip (a duplicated id in the request never causes two DB rows). Preserves input order for stability.

## Frontend behavior

Route `/assets/durations` (mounted inside the assets shell so the sidebar navigation stays consistent).

- Textarea parser splits on any whitespace or comma; trims; drops empties.
- Duplicate detection happens client-side too; the count is displayed under the textarea (e.g. "1668 ids (9 duplicates ignored)").
- `id_type` radio governs input validation display (warns if paste doesn't look like the selected type) but the request always sends the raw list — server matches both columns anyway.
- Submit is disabled when the parsed count is 0 or >5000.
- Histogram buckets: `[0, 60_000), [60_000, 600_000), [600_000, 1_800_000), [1_800_000, 3_600_000), [3_600_000, ∞)`.
- Results table sorts by `duration_ms` desc by default; user can flip.
- CSV export filename: `asset-durations-N.csv` where N is `matched_count`.
