# Tasks - CYB-3004

## Context files
- `backend/internal/usecase/pipeline/usecase.go`
- `backend/internal/usecase/pipeline/usecase_test.go`
- `backend/internal/runtimeos/adapter/interface.go`
- `backend/internal/runtimeos/adapter/argo/adapter.go`
- `backend/internal/runtimeos/adapter/argo/adapter_test.go`
- `backend/cmd/server/core.go`
- `docs/review/api-guide.md`
- `scripts/smoke-runs-dev.sh`

## OpenSpec
- [x] [openspec] Create proposal, design, tasks, decisions, context files, and runtime-os spec delta for adapter-backed Run submit.
- [x] [openspec] Record user approval to proceed past this checkpoint before editing runtime code.

## Implementation
- [x] [backend] Add a submit helper in `pipeline.Usecase` that prefers `RuntimeAdapter.Submit`.
- [x] [backend] Preserve legacy `wfClient.CreateWorkflow` fallback when adapter is unset.
- [x] [backend] Preserve dry-run behavior with no runtime submit call.
- [x] [backend] Preserve runtime config projection owner lookup and cleanup behavior.
- [x] [backend] Preserve status/uid persistence in `PipelineRun` after adapter submit.
- [x] [backend] Preserve existing error semantics for missing runtime backend and submit failures.

## API Contract Sync
- [x] [api] Confirm no OpenAPI shape change is needed.
- [x] [docs] Update `docs/review/api-guide.md` to clarify Run creation submits through RuntimeAdapter.
- [x] [scripts] Confirm existing Run smoke covers list/detail/subresources/unknown-run lifecycle checks; deploy verification will run dry-run or controlled submit separately to avoid accidental dev runtime jobs.
- [x] [openspec] Keep behavior delta aligned with implemented scope.

## Verification
- [x] [backend] `cd backend && go test ./internal/runtimeos/adapter/... ./internal/usecase/pipeline ./internal/handlers/pipeline`.
- [x] [backend] `cd backend && go test ./...`.
- [x] [repo] `git diff --check`.
- [ ] [repo] Non-Terraform pre-commit checks before final commit when the full runtime goal is ready.

## Deploy Verification
- [ ] Deploy backend dev with SHA image and record revision when the full runtime batch is ready for deploy gate.
- [ ] Smoke safe Run API paths and at least one create/dry-run or controlled submit path against Cloud Run dev.
- [ ] If no frontend files changed, skip Chrome MCP per repository rules.

## Push Strategy
- [ ] Do not open or push PRs for this slice alone.
- [ ] Keep work on local `dev`; push once only after the full Run as Kernel goal is complete and verified.
