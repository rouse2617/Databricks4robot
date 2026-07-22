# CYB-3715b decisions

## Why a separate PR (not amended onto #526)

#526 already merged + deployed. Adding contract sync as a distinct PR keeps the runtime landing decoupled from doc-only changes and avoids a force-push to a merged branch. Also matches the memory `feedback_pr_scope_verify_first.md` guidance to split doc-only follow-ups from feat PRs when the feat has already validated.

## Why not describe uuid fields with `format: uuid`

The upstream `mcap_files.device_id` / `collector_id` / `scene_id` are TEXT columns in Postgres — the migration writes them as UUID on assets via `NULLIF(x,'')::uuid` with a per-row exception fallback. In practice a small number of legacy rows may have blank / malformed values that were nulled by the backfill. Advertising them as OpenAPI `format: uuid` would imply strict validation the API does not (and cannot, without a data cleanup) enforce. Left as plain `string` to match the existing McapFile schema (see openapi.yaml line 1253-1259).

## Why not add mcap.<col> filter examples

`mcap.<col>` was already documented pre-CYB-3715. Adding a redundant example row would suggest new users should prefer it over the direct field — the opposite of what we want (direct field is one column read; mcap.<col> is a subquery). The 顶层字段 note calls out that mcap.* still works but is now for legacy paths.

## Why smoke uses `ilike:%`

The smoke assertion needs a value that matches "any non-null row" without hard-coding a specific device model that might disappear on dev. `ilike:%` matches every non-null row, so `total > 0` is stable as long as at least one asset has the column populated. Backfill guarantees this (unless someone truncates the assets table, in which case many other smoke steps would already fail).
