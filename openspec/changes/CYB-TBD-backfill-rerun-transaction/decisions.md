# Decisions — CYB-TBD

## 2026-07-26 — Temporary CYB placeholder

- **Context**: `LINEAR_API_KEY` is not available in this session; the user asked to continue one issue at a time in a worktree and to backfill the Linear id later.
- **Decision**: Use `CYB-TBD-backfill-rerun-transaction` for the branch and OpenSpec dir until a formal CYB id is supplied.
- **Alternatives**: Block until a CYB number exists.
- **Rationale**: OpenSpec docs are local and easy to rename; keeps traceability without blocking the fix.

## 2026-07-26 — Re-queue only after a successful re-submit (transaction boundary)

- **Context**: `Rerun` flipped every matched item to `pending` up front via `PrepareItemsForRerun(itemIDs)`, then looped `UpsertBatchSubtaskRun`. When every upsert failed, `scheduled` stayed empty so the job was never moved to `running` (it stayed terminal), yet all items were already `pending`. `FindSubmittableJobs` only scans `running`/`pilot_running` jobs, so those items dangled forever — never dispatched, never re-failed — and their `error_message` had been wiped by the upfront flip.
- **Decision**: Remove the unconditional upfront batch flip. Re-queue each item (flip to `pending`, clear fields) only after its own `UpsertBatchSubtaskRun` succeeds. On failure the item is left untouched (keeps `failed` status + `error_message`) and recorded in `Skipped`.
- **Alternatives**: Wrap the whole rerun in a DB transaction and roll back on total failure (upsert also creates Argo-facing run rows via the pipeline usecase — not a single-DB-tx boundary, so rollback is not atomic across both stores); or reset item status back to `failed` in a cleanup pass on total failure (more moving parts, still leaves a window).
- **Rationale**: "Success-then-flip" removes the inconsistent window entirely: an item is only `pending` once a run exists for it, so a terminal job never coexists with dangling `pending` items.

## 2026-07-26 — Reset submit_attempts on rerun (parity with DLQ)

- **Context**: `PrepareItemsForRerun` reset `status`/`error_message`/`started_at`/`finished_at` but not `submit_attempts`. The submitter treats `submit_attempts` as a durable, monotonic counter (CYB-3678) and only `ResetFailedItems` (the DLQ retry path) zeroes it. An item poisoned by the attempt cap and retried through `Rerun` kept its exhausted counter, so the first transient submit failure re-poisoned it immediately (zero retry budget).
- **Decision**: Add `submit_attempts = 0` to the `PrepareItemsForRerun` reset UPDATE, matching `ResetFailedItems`. Its only production caller is `Rerun` (an explicit human retry), which is exactly the case that earns a fresh cap.
- **Alternatives**: Zero the counter in a separate call from `Rerun` (extra round-trip, same effect); leave it (the bug).
- **Rationale**: Human retry = fresh attempt budget, identical to the DLQ path.

## 2026-07-26 — Rerun dispatches through the subtaskDeployer seam

- **Context**: `Rerun` called the concrete `uc.pipelineUC.UpsertBatchSubtaskRun`, while the submitter uses the `uc.deployer` (`subtaskDeployer`) interface — which already declares `UpsertBatchSubtaskRun`. The concrete dependency made the rerun transaction boundary untestable without a full transpiler/Argo stack.
- **Decision**: Route `Rerun`'s upsert through `uc.deployer` and gate on `uc.deployer == nil` (equivalent to the old `pipelineUC == nil` fixture guard, since `New` sets `deployer = pipelineUC`).
- **Alternatives**: Keep the concrete call and cover only via integration tests (heavier, needs a DB).
- **Rationale**: Reuses the existing seam and the `fakeDeployer` / `upsertFailingDeployer` fakes; consistent with the submitter.

## 2026-07-26 — Skip manual dev deploy (user request)

- **Context**: The user instructed that these fixes should be pushed as PRs to `dev` without a manual dev deploy; rely on PR CI/CD.
- **Decision**: No Cloud Run dev deploy or Chrome MCP for this backend-only change; provide local build/vet/test evidence in the PR.
- **Alternatives**: Run the full deploy-before-commit gate.
- **Rationale**: Explicit user instruction; the change is backend-only with no API contract or migration.
