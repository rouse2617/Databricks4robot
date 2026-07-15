# CYB-3233 ES sync missing asset fields (storage_uri/thumb_uri/files/mcap_uri)

## Problem
ES asset docs lack `storage_uri`, `thumb_uri`, `files`, and `mcap.mcap_uri`, so
they can't be searched/returned. `builder.go` never emits `thumb_uri`/`files` and
the `mcap` object only sets `recorded_at` (no `mcap_uri`). `storage_uri` IS emitted
(builder.go:57) yet is absent from live docs — the incremental subscriber output
lacked it; a fresh deploy + full reindex (current code) repopulates it. The
canonical index mapping (`init-index.sh`) also doesn't declare these fields.

## Scope
- `searchindex/builder.go`: emit `thumb_uri` (a.ThumbURI), `files` (a.Files) in the
  base doc; add `mcap_uri` (mf.GCSPath, the mcap_uri column) to the `mcap` object.
- `deploy/local/elasticsearch/init-index.sh`: declare `storage_uri` (keyword),
  `thumb_uri` (keyword), `files` (flattened), and `mcap.mcap_uri` (keyword).

## Ops (post-merge on dev)
Recreate the `assets` index with the new mapping (init-index.sh) + reindex, then
verify `storage_uri`/`thumb_uri`/`files`/`mcap.mcap_uri` land in `_source`. This
also confirms whether the emitted-but-missing `storage_uri` was a stale-subscriber
artifact (resolved by the fresh deploy).

## Out of Scope
- Deeper root-cause of why the incremental subscriber previously dropped
  storage_uri (if it recurs after fresh deploy, escalate separately).
