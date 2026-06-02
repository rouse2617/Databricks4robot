# Tasks — CYB-1534 Run Target Model

## Context Files
- `backend/migrations/039_pipeline_tables.sql`
- `backend/internal/models/pipeline.go`
- `backend/internal/repository/pipeline_repository.go`
- `backend/internal/postgres/pipeline_repo.go`
- `backend/internal/usecase/pipeline/usecase.go`
- `backend/internal/handlers/pipeline/handler.go`
- `backend/routes/routes.go`
- `backend/cmd/server/core.go`
- `api/openapi.yaml`
- `docs/review/api-guide.md`
- `scripts/api-guide-smoke.sh`
- `sdk/src/cyber_databrew_sdk/`
- `Frontend/src/api/pipelineApi.ts`

## Checkpoint
- [x] Confirm user approves editing `backend/migrations/`.
- [x] Write OpenSpec proposal, design, tasks, and spec delta.
- [x] Stop for OpenSpec confirmation before runtime edits.

## Implementation
- [x] Add migration for `execution_targets`, `pipeline_runs`, and `pipeline_run_nodes`.
- [x] Extend pipeline models with `PipelineRun`, `PipelineRunNode`, and persisted target fields.
- [x] Add repository interfaces for execution targets, pipeline runs, and run nodes.
- [x] Implement Postgres repositories with JSONB and array scan/marshal coverage.
- [x] Wire repositories in server construction.
- [x] Make `GET /execution-targets` repository-backed with default target bootstrap.
- [x] Add first-class `/pipeline-runs` create/list/get/retry/stop/delete usecase methods.
- [x] Preserve `/deploy`, `/deploy/template`, and `/deployments` compatibility behavior.
- [x] Upsert run nodes from Argo workflow detail during active run refresh.
- [x] Keep `Expired` semantics when Argo Workflow CR is missing after TTL cleanup.

## API Contract Sync
- [x] Update `api/openapi.yaml` schemas and paths.
- [x] Update `docs/review/api-guide.md` examples and error cases.
- [x] Update SDK manager methods and unit tests if public run APIs are exposed.
- [x] Update `scripts/api-guide-smoke.sh` or add a focused run smoke.
- [ ] Update frontend API types for new run and target shapes if consumed.

## Verification
- [x] `cd backend && go test ./internal/usecase/pipeline/...`
- [x] `cd backend && go test ./internal/handlers/pipeline/...`
- [x] `cd backend && go test ./internal/postgres/...`
- [x] `cd backend && go test ./...`
- [x] `cd sdk && uv run pytest tests/unit/ -q` if SDK changes.
- [ ] Apply migration to dev before backend deployment.
- [ ] Smoke execution target list, run create, run get, invalid target, and missing asset behavior on dev.
