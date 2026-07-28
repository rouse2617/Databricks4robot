# Tasks — CYB-TBD

## Context files

- `backend/internal/postgres/backfill_submit_queue.go` — candidate query for submitter jobs.
- `backend/internal/postgres/backfill_submit_queue_integration_test.go` — queue integration coverage.
- `backend/internal/usecase/backfill/submitter.go` — candidate grouping by cluster.
- `backend/internal/usecase/backfill/submitter_cyb3678_test.go` — cluster isolation tests and fake queue/deployer fixtures.
- `backend/internal/usecase/backfill/submitter_test.go` — shared fake submitter queue and deployer.

## Implementation

- [x] [backend] Change `FindSubmittableJobs` to return a target-fair candidate set instead of a purely global oldest-first window.
- [x] [backend] Preserve oldest-first ordering within each target and existing running/`pilot_running` filters.
- [x] [backend] Add a starvation regression where many deferred jobs for target A do not hide a younger healthy target B job.

## API Contract Sync

- [ ] Not applicable: no HTTP route, request, response, status code, or public SDK surface changes.

## Verification

- [x] [backend] Run targeted submitter fairness test.
- [x] [backend] Run targeted Postgres submit queue tests.
- [x] [backend] Run `make fmt && make vet`.
- [x] [backend] Run `go test ./...` before PR.

## Deploy Verification

- [x] User requested no dev deploy; record this as a decision before commit/PR.
- [x] Provide local test evidence in the PR.
