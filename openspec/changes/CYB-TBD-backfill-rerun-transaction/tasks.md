# Tasks — CYB-TBD

## Context files

- `backend/internal/usecase/backfill/usecase.go` — `Rerun`, `rerunFilter`.
- `backend/internal/postgres/backfill_repo.go` — `PrepareItemsForRerun`, `UpdateItemPipelineRun`.
- `backend/internal/postgres/backfill_submit_queue.go` — `ResetFailedItems` (reference semantics), `IncrementItemSubmitAttempts`.
- `backend/internal/usecase/backfill/submitter.go` — `subtaskDeployer` seam, attempt-cap / DLQ path.
- `backend/internal/usecase/backfill/submitter_test.go` — `fakeDeployer`, `upsertFailingDeployer`.
- `backend/internal/usecase/backfill/usecase_test.go` — `trackingBackfillRepo`.

## Implementation

- [x] [backend] `PrepareItemsForRerun` — add `submit_attempts = 0` to the reset UPDATE (parity with `ResetFailedItems`).
- [x] [backend] `Rerun` — call `UpsertBatchSubtaskRun` via `uc.deployer` (subtaskDeployer seam) and gate on `uc.deployer == nil`.
- [x] [backend] `Rerun` — remove the unconditional upfront `PrepareItemsForRerun(itemIDs)` batch flip; re-queue each item only after its upsert succeeds.
- [x] [backend] `Rerun` — on upsert failure, leave the item untouched (preserve `failed` status + `error_message`), record it in `Skipped`.
- [x] [backend] Tests: rerun transaction boundary (all-fail + partial-success) and `PrepareItemsForRerun` SQL surface.

## API Contract Sync

- [x] Not applicable: same `POST /api/v1/backfill/:id/rerun` route, request, response shape, and status codes. Behavior correction only — no OpenAPI / api-guide / SDK change.

## Verification

- [x] [backend] `CGO_ENABLED=0 go build ./...` — passed (cgo disabled on this box).
- [x] [backend] `CGO_ENABLED=0 go vet ./internal/usecase/backfill/... ./internal/postgres/...` — passed.
- [x] [backend] `CGO_ENABLED=0 go test ./internal/usecase/backfill/... ./internal/postgres/...` — passed, incl. new rerun transaction-boundary and `PrepareItemsForRerun` SQL tests.

## Deploy Verification

- [ ] User requested no manual dev deploy; rely on PR CI/CD (see decisions.md). Backend-only, no API contract or migration.
- [ ] Provide local build/vet/test evidence in the PR.
