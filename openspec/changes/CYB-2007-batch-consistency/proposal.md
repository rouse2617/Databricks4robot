# Proposal — CYB-2007

## Why
Batch detail regressions can report a completed 100-item batch as still running, and can show 200 execution records for the same 100 assets. This makes operators distrust batch progress and failure counts after frontend-triggered fan-out runs.

## What Changes

### New Capabilities
- None.

### Modified Capabilities
- Pipeline batch creation SHALL keep one logical backfill item and one visible subtask execution record per submitted asset.
- Pipeline batch progress SHALL converge job status, node summary counts, data coverage, and detail list totals to the same logical batch size.

## Impact
- **Affected code**: `backend/internal/usecase/backfill`, `backend/internal/postgres/backfill_repo.go`, `backend/internal/postgres/pipeline_repo.go`, `backend/internal/usecase/pipeline`
- **New APIs**: none
- **Dependencies**: none

## Scope
- **In scope**: Fix duplicate backfill item/run materialization, batch job terminal status convergence, node summary coverage consistency, and batch detail list count consistency.
- **Out of scope**: New UI controls, new batch rerun modes, database schema changes, and changing public HTTP response shapes.

## Success Criteria
- [ ] Creating a 100-asset batch produces exactly 100 backfill items and 100 visible batch child execution records.
- [ ] A batch with 100 completed and 0 failed items returns `status=completed`, `completedCount=100`, and `failedCount=0`.
- [ ] Node summary reports logical totals consistently: `subtasks.total=100`, pending/running/failure counts match item state, node succeeded count reaches 100, and data coverage is complete.
- [ ] Existing rerun and pilot batch behavior remains covered by tests.
