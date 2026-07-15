# CYB-3232 PATCH /assets/:id supports files + storage_uri/thumb_uri

## Problem
Migration scripts / clients need to set an asset's `files` (algo_input / annot /
delivery URIs), especially on placeholder raw_mcap assets whose files start empty.
Today the only way is direct DB edits. `PATCH /assets/:id` (Update) already writes
the whole row via `repo.Set` and emits `asset_updated` (which also syncs ES), so
adding the fields is a small, proper update entry point.

## Scope
- `handlers/asset/handler.go` Update body: add `files` (map), `storage_uri`,
  `thumb_uri`; pass through to `UpdateInput`.
- `usecase/asset/usecase.go`: `UpdateInput` gains `Files`, `StorageURI`, `ThumbURI`;
  `Update()` merges `files` (same merge semantics as tags) and sets
  storage_uri/thumb_uri when provided, before `repo.Set`.
- API contract sync: openapi + api-guide (+ smoke).

## Semantics / tradeoffs
- `files` uses **merge** semantics (like tags). Full-replace would need a flag.
- Goes through Update → emits `asset_updated` → files also flow to ES.
- Does not fix "placeholder starts with empty files" at the root, but gives a
  proper API update path (better than direct DB edits).

## Out of Scope
- No schema change (columns exist), no migration.
