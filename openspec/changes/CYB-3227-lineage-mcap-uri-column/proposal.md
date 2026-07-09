# CYB-3227 Fix upstream lineage using wrong column (storage_uri → mcap_uri)

## Problem

`GET /assets/{id}/lineage` (and provenance's embedded lineage) always shows
**no upstream MCAP** — `upstream` is `{}` for every asset.

Root cause: the upstream query in `backend/internal/handlers/asset/lineage_response.go:50`
selects a column that does not exist on `mcap_files`:

```sql
SELECT mcap_file_id, COALESCE(storage_uri,''), COALESCE(ingest_state,'')
FROM mcap_files
WHERE mcap_file_id = (SELECT mcap_file_id FROM assets WHERE asset_id = $1)
```

- `mcap_files` has column **`mcap_uri`** (baseline migration:
  `"mcap_uri" text NOT NULL DEFAULT ''`). `storage_uri` is a column on `assets`,
  not `mcap_files`.
- So the query fails every call with `column "storage_uri" does not exist`
  (SQLSTATE 42703). The `if err == nil && mcapFileID != ""` guard (line 54) then
  skips populating `out.Upstream`, leaving it `{}`.
- Unlike the algo/delivery/eval queries in the same function, this query has **no
  `slog.Warn` on error**, so it has been failing silently.

Tests missed it: the upstream block runs under `h.pg` (real `*postgres.Client`,
`if h.pg != nil`), while the existing `buildLineageResponse` tests only wire the
`h.pgq` mock and leave `h.pg == nil`, so the upstream query is never exercised.

## Scope

- `lineage_response.go`: `storage_uri` → `mcap_uri` in the upstream query.
- Add a `slog.Warn` when the upstream query errors (consistent with the other
  queries in the function), so this class of failure surfaces instead of silently
  returning empty upstream.

## Out of Scope

- No HTTP API response-shape change — the JSON field is already `mcap_uri`
  (`out.Upstream["mcap_uri"]`), only the SQL source column was wrong.
- No schema change / migration.
- Refactoring the upstream query to use the mockable `h.pgq` querier (for unit
  testability) is noted but deferred to keep the fix minimal; verification is via
  dev curl of `/assets/{id}/lineage`.
