# Proposal — CYB-3285

## Why

The MCAP 文件 list (`McapFilesPage`) shows 通道数 (`channel_count`) + 分块数
(`chunk_count`) columns. For grace-ingested mcaps these are never populated and
render as「—」on every row. Users don't want them; what's useful is **how many
child segments each mcap has** (segments are cut from an mcap and carry its
`mcap_file_id`).

## What Changes

### Modified Capabilities
- **storage — mcap file list** — the mcap-files list response gains
  `segment_count`: the number of `asset_type='segment'` assets whose
  `mcap_file_id` is that mcap (non-deleted). Computed in the list query via a
  correlated subquery (single query, no N+1, no migration).
- **frontend — mcap files page** — replace the 通道数 + 分块数 columns with a
  single **子 segment 数** column (`dataIndex: segment_count`).

### Design decisions
- **Count `asset_type='segment'` only** — actions/clips/frames/tasks also inherit
  the parent's `mcap_file_id`, so counting all assets would over-count.
- **Keep `channel_count`/`chunk_count` in the model/DB** — only remove them from
  the UI. No schema change; the fields remain for any future use.
- **Correlated subquery, not a JOIN** — avoids `mcap_file_id` ambiguity in the
  existing SELECT and keeps the WHERE/scan changes minimal.

## Impact
- **Affected code**:
  - `backend/internal/models/*` — `McapFile` gains `SegmentCount int64` (`json:"segment_count"`).
  - `backend/internal/postgres/repos.go` — `McapFileRepo.List` (+ `Get`) SELECT adds the correlated count + scan target.
  - `api/openapi.yaml` + `docs/review/api-guide.md` — document `segment_count` on the McapFile schema / mcap list.
  - `Frontend/src/api/types.ts` — `McapFile` gains `segment_count?: number`.
  - `Frontend/src/pages/McapFilesPage.tsx` — column swap.
- **New APIs**: none (additive field on an existing response).
- **Dependencies / Migration**: none.
- **SDK**: no change — `storage.list_files` is a dict passthrough; the new field flows through.

## Scope
- **In scope**: backend `segment_count` on mcap list/get + the frontend column swap + API contract sync (OpenAPI, api-guide).
- **Out of scope**: removing `channel_count`/`chunk_count` columns from the DB/model; backfilling channel/chunk metadata; any counting of non-segment child assets.

## Success Criteria
- [ ] `GET /api/v1/mcap-files` returns `segment_count` per item = # of non-deleted `asset_type='segment'` with that `mcap_file_id`.
- [ ] MCAP list page shows a 子 segment 数 column; 通道数 + 分块数 columns are gone.
- [ ] A mcap with N segments shows N; a mcap with 0 shows 0.
- [ ] `go build ./...` + touched pkg tests pass; `npm run build` clean; no console errors on the list page.
