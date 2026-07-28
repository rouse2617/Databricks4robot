# CYB-4305 tasks

## Backend

- [ ] `models/asset_lineage.go` — new file for batch-lineage types (`AssetLineageBatchRequest`, `AssetLineageBatchResponse`, `LineageBatchItem`, `LineageBatchStats`). Item struct has pointer fields for parent/root/logical/upstream/downstream so nulls serialize correctly.
- [ ] `postgres/repos.go` — new `AssetRepo.LookupLineage(ctx, assetIDs []string) ([]LineageRow, error)`. Single-SELECT reads `asset_id, grace_video_id, parent_asset_id, root_asset_id, logical_asset_id, is_current, COALESCE(revision,0)`. Bind ids via `ANY($1)` on both columns. Guard `is_deleted = FALSE`.
- [ ] `searchindex/lineage_batch.go` (create if the codebase groups ES helpers there; otherwise add to existing ES client) — new `LineageDocsByAssetID(ctx, assetIDs []string) (map[string]LineageProjection, error)`. Uses ES `_mget` with `_source: ["lineage_upstream_ids", "lineage_downstream_ids", "lineage_relation_types"]`.
- [ ] `usecase/asset/usecase.go` — new `LookupLineage(ctx, req AssetLineageBatchRequest) (*AssetLineageBatchResponse, error)`:
  - Dedup ids preserving first occurrence
  - Reuse the existing `resolveAssetIDs` helper (introduced for CYB-4306) to resolve grace_video_id → asset_id set
  - Call `LookupLineage` repo method for depth=1 rows
  - If depth="all", also call the ES projection lookup for the same asset_id set, merge into items
  - Compute stats (has_parent_count, is_root_count where root=self, orphan_count where parent+root both null, is_current_count, relation_type_counts as `map[string]int` from the depth=all responses)
  - Preserve input_id → item mapping in insertion order
- [ ] `handlers/asset/handler.go` — new `Handler.LookupLineage(c *gin.Context)`:
  - Decode + validate: ids non-empty (400 `ID_LIST_REQUIRED`), ≤5000 (400 `ID_LIST_TOO_LARGE` — reuse), depth ∈ `{"", "1", 1, "all"}` (400 `INVALID_LINEAGE_DEPTH` — new code)
  - Normalize `depth` param: JSON accepts both `1` and `"1"` for depth=1; anything else → "all"
- [ ] `routes/routes.go` — mount `POST /assets/lineage-batch` in the assets:read scope group next to `/assets/durations` and `/assets/costs`
- [ ] Tests:
  - Repo test with fakeDB: SQL args + row parse; guards is_deleted
  - Usecase test: depth=1 path (no ES call); depth=all path with ES mock; missing_ids populated; is_root_count when root=self; orphan_count for null-parent-null-root
  - Handler test: 200 with mock data + all validation branches

## Frontend

- [ ] `api/assets.ts` — add `AssetLineageBatchRequest`, `AssetLineageBatchResponse`, `LineageBatchItem`, `LineageBatchStats` types + `assetsApi.lookupLineage(req)` client
- [ ] `pages/presets/lineage.tsx` — preset config:
  - `filters.initial = { depth: 1 }`, `filters.render` renders a Segmented with icons for the two depth options
  - `filters.validate` returns null (nothing to validate)
  - `fetch({ids, id_type, filters}) → assetsApi.lookupLineage({ ids, id_type, depth: filters.depth })`
  - `columns` — a callback that returns 7 cols for depth=1 and 10 cols for depth=all (matches CYB-4306 pattern: `preset.columns` accepts a callback)
  - `histogram(items)` — returns categorical bars of `relation_types` distribution when depth=all; returns undefined (skipped) when depth=1
  - `extraStats(response)` — 5 entries: 已匹配 (matched_count), 有 parent 的 (has_parent_count), 是 root 的 (is_root_count), 孤儿 (orphan_count), 当前版本 (is_current_count)
  - `csv.filename(n, filters?)` returns `asset-lineage-depth${filters?.depth ?? 1}-${n}.csv`
  - `csv.header` and `csv.row` switch on the current depth
- [ ] `pages/presets/index.tsx` — register the new preset:
  - Extend `BatchLookupPresetKey` to `"durations" | "costs" | "lineage"`
  - Add `lineage: lineagePreset` to `BATCH_LOOKUP_PRESETS`
  - Add `{ key: "lineage", label: "血缘", icon: <PartitionOutlined /> }` to `PRESET_OPTIONS`
  - Icon choice: `PartitionOutlined` (Ant Design) is the natural fit for lineage/graph
- [ ] Frontend tests: `presets/lineage.test.tsx` covering column set switches on depth, histogram only appears for depth=all, csv filename encodes depth, extraStats counts are computed correctly

## Verification

- [ ] `cd backend && go test ./...`
- [ ] `cd backend && go build -ldflags "-w -s" ./...`
- [ ] `cd Frontend && npx tsc --noEmit`
- [ ] `cd Frontend && npx vitest run <touched files>`
- [ ] `cd Frontend && npx vite build`
- [ ] Grep-check: `PartitionOutlined` import in `presets/index.tsx`; `lineage` present in all three registry constants

## Ship

- [ ] Commit + push + PR base=dev
- [ ] Merge + wait deploy
- [ ] Dev smoke on a real 100-id sample at both depths; verify Segmented switching depth refetches
