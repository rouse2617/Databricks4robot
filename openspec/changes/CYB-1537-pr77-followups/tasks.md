# Tasks — CYB-1537 PR #77 Review Follow-up

## Context Files
- `backend/internal/handlers/pipeline/handler.go`
- `backend/internal/handlers/workflow/logs_sse.go`
- `backend/internal/usecase/pipeline/usecase.go`
- `backend/internal/usecase/pipeline/usecase_test.go`
- `backend/internal/handlers/pipeline/handler_test.go`
- `backend/internal/handlers/workflow/handler_test.go`
- `backend/internal/repository/pipeline_repository.go`
- `backend/internal/postgres/pipeline_repo.go`

## Checkpoint
- [x] Write OpenSpec proposal, tasks, spec delta, and decisions.
- [x] Stop for OpenSpec confirmation before runtime edits.

## Implementation
- [ ] Fix 1: `CreateRunByTemplate` returns 400 on malformed JSON body while keeping empty-body support.
- [ ] Fix 2: `getWorkflowResourceUsage` looks up via `pipelineRepo.FindByWorkflowName` first; falls back to `deploymentRepo.FindAll` only on miss.
- [ ] Fix 3: `streamWorkflowLogs` uses `len(line)` and `line[:remaining]` instead of `[]byte(line)` round-trips.

## Tests
- [ ] New test: `CreateRunByTemplate` returns 400 on malformed JSON.
- [ ] New test: `CreateRunByTemplate` succeeds on empty body (regression guard).
- [ ] New test: `getWorkflowResourceUsage` calls `pipelineRepo.FindByWorkflowName` and does not call `deploymentRepo.FindAll` when the run exists.
- [ ] New test: `getWorkflowResourceUsage` falls back to `deploymentRepo.FindAll` when the run does not exist.
- [ ] New test or assertion: `streamWorkflowLogs` byte accounting matches `len(line)` after the change (regression guard for the `[]byte` removal).

## API Contract Sync
- N/A — no public HTTP API shape changes. `CreateRunByTemplate` still accepts an optional body; the 400 on malformed body matches the existing `CreateRun` handler's behavior and reuses the same `httpresp.BadRequest` envelope.

## Verification
- [ ] `cd backend && make fmt && make vet`
- [ ] `cd backend && go test ./internal/handlers/pipeline/...`
- [ ] `cd backend && go test ./internal/handlers/workflow/...`
- [ ] `cd backend && go test ./internal/usecase/pipeline/...`
- [ ] `cd backend && go test ./...`
- [ ] Dev smoke: `POST /api/v1/pipeline-runs/template/{id}` with body `not-json` returns 400.
- [ ] Dev smoke: `GET /api/v1/workflows/{name}/resources` returns same shape as before (no behavior change visible to the caller).
