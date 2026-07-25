# Tasks — CYB-TBD

## Context files

- `backend/internal/postgres/backfill_submit_queue.go` — submittable job SQL projection and scan order.
- `backend/internal/postgres/backfill_submit_queue_integration_test.go` — integration coverage for job selection fields.
- `backend/internal/usecase/backfill/submitter.go` — `DeployOptions.Owner` propagation from selected job.
- `backend/internal/usecase/backfill/submitter_test.go` — submitter fake deployer captures deployment options.

## Implementation

- [x] [backend] Update `FindSubmittableJobs` to select and scan the batch job creator.
- [x] [backend] Add repository coverage proving `CreatedBy` survives the submit queue read path.
- [x] [backend] Add submitter coverage proving `CreatedBy` becomes `DeployOptions.Owner`.

## API Contract Sync

- [ ] Not applicable: no HTTP route, request, response, status code, or public SDK surface changes.

## Verification

- [x] [backend] Run targeted Postgres submit-queue integration test.
- [x] [backend] Run targeted backfill submitter tests.
- [x] [backend] Run `make fmt && make vet`.
- [x] [backend] Run `go test ./...` before PR.

## Deploy Verification

- [x] User requested no dev deploy; record this as a decision before commit/PR.
- [x] Provide local test evidence in the PR.
