# Tasks — CYB-2767

## Implementation

- [ ] `[backend]` Add per-item timeout to `processBatchJob` in `backend/internal/usecase/pipeline/batch.go`: wrap `CreateRunByTemplateID` call with `context.WithTimeout` (60s), send timeout error as item failure
- [ ] `[backend]` Add per-item timeout to `runItems` in `backend/internal/usecase/backfill/usecase.go`: wrap `executeItem` call with `context.WithTimeout` (60s), log timeout as warning
- [ ] `[backend]` Ensure timeout value is consistent: 60 seconds for both paths

## Deploy Verification

- [ ] `[backend]` Deploy to Cloud Run dev via `deploy/cloudrun/backend-dev.sh`
- [ ] `[backend]` Navigate to the hung batch `batch_5cd83729...` and verify it eventually transitions to terminal state (or items show as failed)
- [ ] `[backend]` Create a new test batch with multiple assets and verify it completes normally

## Context files

- `backend/internal/usecase/pipeline/batch.go` — `processBatchJob`, worker loop
- `backend/internal/usecase/backfill/usecase.go` — `runItems`, `executeItem`
