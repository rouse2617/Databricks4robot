# Tasks — CYB-4450

## Frontend

### Delete
- [x] `Frontend/src/pages/AssetDetailPage.tsx` — remove `AlgoTab` lazy import, `parseAlgoResults` function, `algoEvents` / `algoEventsCursor` / `algoEventsLoading` state, `loadAlgoEvents` callback, the `algoEvents` reseed in reset effect, `[id, loadAlgoEvents, ...]` dep array entry, the `algo` tab item from `tabItems`, the `algo` key from `ASSET_DETAIL_TAB_KEYS`
- [x] `Frontend/src/components/asset-detail/AlgoTab.tsx` — file deleted (no callers left)

### Rename
- [x] `Frontend/src/pages/AssetDetailPage.tsx` — the existing `runs` tab label `运行历史` → `运行记录`

## Backend

None. The CYB-4297 endpoint `GET /api/v1/assets/:id/runs` (implemented in `pipelineHandler.ListRunsByAsset`) already does the reverse-lookup. The existing `RunsTab` (`/components/asset-detail/RunsTab.tsx`) already calls it via `listPipelineRunsByAsset`.

## API contract sync

None — no new endpoint, no schema change. RunsTab's `PipelineRunListResponse` shape was synced in CYB-4297.

## Verification (done)

- [x] `cd Frontend && npm run build` — succeeds
- [x] `npx biome check src/pages/AssetDetailPage.tsx` — clean
- [x] `cd backend && go build ./...` — succeeds (no backend changes; verified)
