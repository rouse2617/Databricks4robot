# CYB-4294 decisions

## Why POST not GET

id lists can hit 5000; a GET query string blows past URL length limits and gets truncated by proxies/logs. POST body carries the list cleanly and keeps ids out of access logs (grace_video_id has weak PII footprint but sitting in every request log for months is unnecessary).

## Why one SQL statement, not two

Splitting into "match by asset_id" + "match by grace_video_id" doubles the round trip and doubles the index scan. The single `WHERE asset_id = ANY($1) OR grace_video_id = ANY($1)` runs each ANY as an index scan (asset_id PK, grace_video_id partial index from CYB-4011) then unions — cheaper. Also lets us support mixed-id input transparently, which is the top user complaint.

## Why 5000 cap

- Peak observed real batch on dev: 1668 ids
- Safety margin: 3× peak
- SQL: `ANY(text[])` with 5000 elems is fine; both indexes handle it
- Response payload: 5000 items × ~100 bytes each ≈ 500 KB, well under any transport limit
- Above 5000 we'd want streaming or job-based rather than sync request/response

Caller can split larger sets and stitch client-side.

## Why `filtered_out_ids` is separate from `missing_ids`

They mean different things to the user:
- `missing_ids` — "these do not exist in the platform, check your input or the ingest pipeline"
- `filtered_out_ids` — "these exist but do not match your duration window, they're just not what you wanted"

Merging them would obscure a real diagnostic signal (missing usually means an upstream problem worth chasing).

## Why compute stats server-side

Percentiles on 5000 sorted numbers is trivial in Go and avoids shipping the same numbers back to every dashboard tab that recomputes them. Keeps the frontend a display layer.

## Why frontend histogram is client-side

The full items array is already on the client for the table. Rebucketing server-side would double the data path for no benefit. Buckets are hard-coded for the reasons in the tasks doc — no config surface until a second real request exists.

## Why `id_type` is a UI hint only

The server always matches both columns via OR — it costs nothing extra given the two indexes and it lets a mixed paste "just work". Making the server honor `id_type` would just be a client-facing footgun (user typos the radio → half their ids silently miss).

## Why reuse `assets:read` scope

Duration is already visible via `GET /assets/:id` under `assets:read`. This endpoint just batches. No new scope needed; anyone who can read one asset can read many at once.
