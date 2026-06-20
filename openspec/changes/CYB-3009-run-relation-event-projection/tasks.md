# Tasks - CYB-3009

## Context files

- `backend/internal/usecase/pipeline/usecase.go`
- `backend/internal/usecase/pipeline/usecase_test.go`
- `backend/internal/models/pipeline.go`
- `backend/internal/runtimeos/state/state_machine.go`

## OpenSpec

- [x] [openspec] Create proposal, design, tasks, decisions, context files, and runtime-os spec delta.
- [x] [openspec] Continue without stopping for another checkpoint per user instruction.

## Implementation

- [x] [backend] Add source Run event payloads for child Run creation.
- [x] [backend] Project event-derived child relations in `ListRunChildren`.
- [x] [backend] Preserve batch child relation projection.
- [x] [backend] Add tests for rerun/resubmit relation projection.
- [x] [api/docs] Document event-derived `retry_of` / `resubmit_of` / `rerun_of` relation values.

## Verification

- [x] [backend] `cd backend && go test ./internal/usecase/pipeline`.
- [x] [backend] Include this path in the final full backend test pass: `cd backend && go test ./...`.
- [ ] [repo] Non-Terraform pre-commit checks before final commit when the full runtime goal is ready.

## Deploy Verification

- [ ] Deploy backend dev when the full runtime batch is ready for deploy gate.
- [ ] Smoke `/runs/{id}/children` for a controlled rerun/resubmit relation when safe.

## Push Strategy

- [ ] Do not open or push a PR for this slice alone.
- [ ] Keep work on local `dev`; push once only after the full Run as Kernel goal is complete and verified.
