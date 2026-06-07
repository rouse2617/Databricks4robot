# Tasks — CYB-1798

## Context Files
- `backend/internal/usecase/pipeline/usecase.go` — duration helper and cost summary composition.
- `backend/internal/usecase/pipeline/usecase_test.go` — focused backend regression coverage.

## Implementation
- [x] [backend] Treat zero `startedAt` or `finishedAt` as unavailable duration.
- [x] [backend] Preserve valid duration behavior.

## Verification
- [x] [backend] Add focused duration helper regression tests.
- [x] [backend] `go test ./internal/usecase/pipeline`
- [x] [backend] `go test ./...`

## Documentation / Tracking
- [ ] PR body references `CYB-1798`.
- [ ] PR body references `openspec/changes/CYB-1798-pipeline-duration-zero-time`.

## Verification Evidence
- 2026-06-07: `cd backend && go test ./internal/usecase/pipeline` passed.
- 2026-06-07: `cd backend && go test ./...` passed.
