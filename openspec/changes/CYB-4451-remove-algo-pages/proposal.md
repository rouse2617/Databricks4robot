# Proposal — CYB-4451: Remove legacy /algo + /algo-runs frontend

## Why

Following CYB-4450 (asset detail's 「算法处理」 tab removal), the top-level sidebar still exposes 「算法运行」 (`/algo-runs`) and 「算法处理」 (`/algo`), which read CF-era `algo_runs` data that is no longer maintained. The platform has migrated to pipeline templates; those legacy CF views no longer represent the system. The asset-detail tab was removed in PR #630; this PR finishes the front-end cleanup.

## What Changes

### Modified Capabilities
- **app-routing**: `/algo`, `/algo-runs`, `/algo-runs/:run_id` redirect to canonical `/runs` surface; pages deleted.
- **app-sidebar**: 「算法运行」 + 「算法处理」 entries removed.
- **asset-run-id-link**: popover re-points from `algoRunsApi` (`AlgoRun` type) → `getPipelineRun` (`PipelineRun`).

### New Capabilities
- None.

## Impact
- **Affected code**:
  - `Frontend/src/App.tsx` — 3 lazy imports removed; 3 routes replaced with `<Navigate replace>`
  - `Frontend/src/components/AppLayout.tsx` — 2 menu items + active-route detection + breadcrumb override + 2 icon imports removed
  - `Frontend/src/components/asset-detail/RunIdLink.tsx` — `algoRunsApi.get` → `getPipelineRun`; popover title `算法运行` → `运行`
- **Deleted files**:
  - `Frontend/src/pages/AlgoRunsPage.tsx`
  - `Frontend/src/pages/AlgoRunDetailPage.tsx`
  - `Frontend/src/pages/AlgoProcessingPage.tsx`
  - `Frontend/src/api/algoRuns.ts`
  - `Frontend/src/api/algoRuns.test.ts`
- **New APIs**: None
- **Dependencies**: None

## Scope
- **In scope**:
  - Delete the 3 legacy pages + the `algoRuns` API client and its test
  - Replace the 3 routes with `<Navigate>` to `/runs` (bookmarks)
  - Re-point RunIdLink popover from `AlgoRun` to `PipelineRun`
  - Drop 「算法运行」 / 「算法处理」 sidebar entries

- **Out of scope**:
  - Backend `handlers/algorun/handler.go` + `handlers/asset/algo_handler.go` — separate PR (SDK + future scripts may still call them)
  - OpenAPI paths under `[AlgoRuns]` in `api/openapi.yaml` — backend still serves the endpoints
  - `docs/review/api-guide.md §2.0` — backend reference still valid while the endpoints exist
  - `Frontend/src/lib/algoStatus.ts` — still used by `AlgoMatrixGrid`, `useRetryAllFailed`, `lib/statusColor.ts`

## Success Criteria
- [ ] 「算法运行」 + 「算法处理」 entries no longer appear in the left sidebar
- [ ] Direct visits to `/algo` and `/algo-runs` auto-redirect to `/runs`
- [ ] Direct visits to `/algo-runs/:run_id` auto-redirect to `/runs/:run_id`
- [ ] Asset detail page's run-id popovers still work (now sourced from `PipelineRun`)
- [ ] `npm run build` succeeds; no dead imports
