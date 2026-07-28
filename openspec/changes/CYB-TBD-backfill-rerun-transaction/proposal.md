# Proposal — CYB-TBD

## Why

The batch **Rerun** entry point (`POST /api/v1/backfill/:id/rerun` →
`backfill.Usecase.Rerun`) re-queues matched items by calling
`PrepareItemsForRerun` and then re-submitting each item. Two defects make the
retry unsafe:

1. **`submit_attempts` is never reset on rerun.** An item that exhausted its
   transient-retry budget (CYB-3678) is parked in the DLQ (`status=failed`) with
   `submit_attempts` at the cap. `PrepareItemsForRerun` clears
   `status/error_message/started_at/finished_at` but **not** `submit_attempts`.
   The DLQ path (`ResetFailedItems`) explicitly resets it ("an explicit human
   retry earns a fresh cap"); rerun does not. So a reran item is flipped to
   `pending` with a maxed-out counter — the very first transient submit error
   in the submitter calls `IncrementItemSubmitAttempts`, immediately re-crosses
   the cap, and re-poisons the item to `failed`. The retry has **zero** budget.

2. **Rerun leaves job/item state inconsistent when every submit fails.** Rerun
   first flips **all** runnable items to `pending` in one batch
   (`PrepareItemsForRerun(itemIDs)` — this also wipes their `error_message`),
   then loops `UpsertBatchSubtaskRun` per item. Only if at least one item was
   scheduled does it set the job to `running`. If **all** upserts fail,
   `scheduled` is empty, the job stays terminal (`failed`), but every item is
   now `pending`. The submitter's `FindSubmittableJobs` only picks
   `running`/`pilot_running` jobs, so those `pending` items are never
   dispatched and never re-failed — dangling forever, with their original
   failure reason erased.

## What Changes

### Modified Capabilities

- **backfill**: `PrepareItemsForRerun` also resets `submit_attempts = 0`, so a
  human rerun earns a fresh transient-retry budget — matching the DLQ
  `ResetFailedItems` semantics. `PrepareItemsForRerun` is only called from the
  rerun path, so this does not affect automatic dispatch attempt accounting.
- **backfill**: `Rerun` re-queues an item **only after** its new run is created.
  Each item is upserted first; on success it is re-queued (flipped to `pending`,
  fields cleared) and bound to the run; on failure it is left untouched
  (original `failed` status and `error_message` preserved). When every submit
  fails, no item is flipped, so the job never ends up terminal-with-pending
  items.
- **backfill (internal)**: `Rerun` now calls `UpsertBatchSubtaskRun` through the
  `subtaskDeployer` seam (`uc.deployer`) already used by the submitter, instead
  of the concrete `uc.pipelineUC`. This makes the transaction boundary unit
  testable with the existing fake deployers and keeps rerun consistent with the
  submitter's dependency shape.

## Impact

- **Affected code**:
  - `backend/internal/postgres/backfill_repo.go` (`PrepareItemsForRerun` SQL)
  - `backend/internal/usecase/backfill/usecase.go` (`Rerun`)
- **Tests**:
  - `backend/internal/usecase/backfill/usecase_test.go` (rerun transaction boundary)
  - `backend/internal/postgres/backfill_repo_rerun_test.go` (SQL surface — new)
- **New APIs**: None. Same HTTP endpoint and response shape; behavior fix only.
- **Dependencies / migration**: None. `submit_attempts` column already exists.

## Scope

- **In scope**: reset `submit_attempts` on rerun; re-queue only successfully
  re-submitted items; preserve failed items' status/reason when submit fails;
  route rerun's upsert through the `subtaskDeployer` seam for testability.
- **Out of scope**: DLQ `ResetFailedItems` (already correct); single-run
  priority passthrough (P2-7); DLQ failure-reason loss on the submitter's
  `runID != ""` branch (P2-6) — tracked separately.

## Success Criteria

- [ ] When every rerun submit fails, the job stays terminal and matched items
      keep their `failed` status and `error_message` (no dangling `pending`).
- [ ] On partial success, only successfully re-submitted items become `pending`
      and bind to a run; the job moves to `running`; failed items are untouched.
- [ ] `PrepareItemsForRerun` resets `submit_attempts = 0` alongside the existing
      field resets.
