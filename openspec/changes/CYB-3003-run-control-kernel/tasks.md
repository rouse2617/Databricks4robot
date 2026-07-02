# Tasks - CYB-3003

## Context files
- `backend/internal/runtimeos/run/service.go`
- `backend/internal/runtimeos/run/service_test.go`
- `backend/internal/runtimeos/adapter/interface.go`
- `backend/internal/runtimeos/adapter/argo/adapter.go`
- `backend/internal/runtimeos/adapter/argo/adapter_test.go`
- `backend/internal/usecase/pipeline/usecase.go`
- `backend/internal/usecase/pipeline/usecase_test.go`
- `backend/internal/handlers/pipeline/handler.go`
- `backend/internal/handlers/pipeline/handler_test.go`
- `backend/cmd/server/core.go`
- `backend/internal/argo/client.go`
- `api/openapi.yaml`
- `docs/review/api-guide.md`
- `scripts/smoke-runs-dev.sh`

## OpenSpec
- [x] [openspec] Create proposal, design, tasks, decisions, context files, and runtime-os spec delta for Run lifecycle control.
- [x] [openspec] Record user approval to proceed past this checkpoint before editing runtime code.

## Implementation
- [x] [backend] Add a Run Kernel lifecycle-control contract or helper that can invoke runtime adapter operations from stable Run identity.
- [x] [backend] Wire ArgoRuntimeAdapter into the production Run lifecycle-control path without changing public routes.
- [x] [backend] Preserve legacy workflow-client fallback for direct unit-test construction and old compatibility paths.
- [x] [backend] Keep RunEvent ledger requested/succeeded/failed semantics for retry, stop, suspend, resume, and terminate.
- [x] [backend] Ensure missing workflow/runtime refs return existing invalid-argument style errors and do not call the adapter.
- [x] [backend] Keep legacy `/pipeline-runs/:id/retry` full rerun semantics separate from `/runs/:id/retry` runtime retry semantics.

## API Contract Sync
- [x] [api] Confirm `api/openapi.yaml` already documents existing Run lifecycle operation paths and update descriptions if needed.
- [x] [docs] Update `docs/review/api-guide.md` to clarify Run lifecycle controls are product operations and Workflow controls are runtime debug.
- [x] [scripts] Update or confirm `scripts/smoke-runs-dev.sh` covers product Run operation error paths without creating unsafe dev side effects.
- [x] [openspec] Keep behavior delta aligned with implemented scope.

## Verification
- [x] [backend] `cd backend && go test ./internal/runtimeos/run ./internal/runtimeos/adapter/... ./internal/usecase/pipeline ./internal/handlers/pipeline`.
- [x] [backend] `cd backend && go test ./...`.
- [x] [scripts] `bash -n scripts/smoke-runs-dev.sh`.
- [x] [repo] `git diff --check`.
- [ ] [repo] Non-Terraform pre-commit checks before commit when runtime code is complete.

## Deploy Verification
- [ ] Deploy backend dev with SHA image and record revision.
- [ ] Smoke `/readyz`, `/api/v1/runs`, `/api/v1/runs/:id/runtime`, and safe lifecycle error paths against Cloud Run dev.
- [ ] If no frontend files changed, skip Chrome MCP per repository rules.

## Push Strategy
- [ ] Do not open or push PRs for this slice alone.
- [ ] Keep work on local `dev`; push once only after the full Run as Kernel goal is complete and verified.
