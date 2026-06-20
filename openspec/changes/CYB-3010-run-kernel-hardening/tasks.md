# Tasks - CYB-3010

## Context files
- `backend/migrations/040_pipeline_run_model.sql`
- `backend/migrations/046_pipeline_run_events.sql`
- `backend/migrations/052_batch_job_pipeline_runs.sql`
- `backend/internal/models/pipeline.go`
- `backend/internal/repository/pipeline_repository.go`
- `backend/internal/postgres/pipeline_repo.go`
- `backend/internal/usecase/pipeline/usecase.go`
- `backend/internal/runtimeos/state/state_machine.go`
- `Frontend/src/api/runApi.ts`
- `Frontend/src/pages/BatchJobDetailPage.tsx`

## OpenSpec
- [x] [openspec] Create proposal, design, tasks, decisions, context files, and runtime-os spec delta.
- [x] [openspec] Record user instruction to continue without another checkpoint prompt.
- [x] [openspec] Record explicit migration scope approval.

## Implementation
- [x] [backend] Add additive migration for `run_relations` and `run_inputs`.
- [x] [backend] Add Run relation/input models and repository contracts.
- [x] [backend] Implement Postgres upsert/list methods for durable Run relations and inputs.
- [x] [backend] Ensure BatchJob creation/upsert creates a parent Run root.
- [x] [backend] Persist `batch_child` relations for new child Runs.
- [x] [backend] Persist rerun, resubmit, and retry-created child relations.
- [x] [backend] Materialize asset Run inputs at Run creation time.
- [x] [backend] Materialize config Run inputs at Run creation time using existing sanitized config metadata.
- [x] [backend] Materialize runtime target and parameter Run inputs at Run creation time.
- [x] [backend] Make `/runs/{id}/children` prefer durable relations and fall back to legacy batch/event projections.
- [x] [backend] Make `/runs/{id}/inputs` prefer durable inputs and fall back to legacy projections.
- [x] [backend] Add generic diagnostic reasons for Failed/Error/Expired with empty messages if missing.
- [x] [backend] Add aggregate health fields for running batches with child failures/blocking.
- [x] [Frontend] Surface aggregate health without changing the Batch Detail route or primary layout.

## API Contract Sync
- [x] [api] Update `api/openapi.yaml` if aggregate health response fields change.
- [x] [docs] Update `docs/review/api-guide.md` to describe durable Run relations and inputs.
- [x] [sdk] Regenerate or update SDK models/tests if OpenAPI response fields change.
- [x] [scripts] Update `scripts/smoke-runs-dev.sh` for children/input smoke where safe.
- [x] [openspec] Keep runtime-os spec delta aligned with implemented behavior.

## Verification
- [x] [backend] `cd backend && go test ./internal/postgres ./internal/usecase/pipeline ./internal/runtimeos/state ./internal/handlers/pipeline`.
- [x] [backend] `cd backend && go test ./...`.
- [x] [Frontend] Run targeted tests for Batch Detail and Run API type changes.
- [x] [Frontend] `cd Frontend && npm run build`.
- [x] [sdk] If SDK touched: `cd sdk && uv run ruff check src/ tests/unit/ && uv run pytest tests/unit/ -q`.
- [x] [repo] `git diff --check`.

## Deploy Verification
- [x] [db] Apply `backend/migrations/056_run_kernel_facts.sql` to dev before backend deploy.
- [x] [backend] Deploy backend dev with SHA image and record revision.
- [x] [backend] Smoke `/readyz`, `/api/v1/runs`, `/api/v1/runs/{id}/inputs`, and `/api/v1/runs/{id}/children`.
- [x] [Frontend] Start local frontend against dev backend and verify Batch Detail aggregate health and Run Inspector metadata.

### Deploy record
| Service | Image tag | Cloud Run revision | URL |
|---------|-----------|-------------------|-----|
| backend-dev | `cyber-databrew-backend:bba96db-cyb3010-run-facts-20260620210151` | `cyber-databrew-backend-dev-00972-7rv` | `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app` |

### Verification evidence
- Migration applied to dev: `backend/migrations/056_run_kernel_facts.sql`.
- Backend smoke: `scripts/smoke-runs-dev.sh` -> `23 passed, 0 failed`.
- API spot checks:
  - `/readyz` -> 200.
  - `/api/v1/runs/3224bbdd-121a-40a5-ba7f-568be2b6a69a/children` -> 200, `total=2`, `healthStatus=degraded`, `relations=2`, no zero-value timestamps.
  - `/api/v1/runs/3224bbdd-121a-40a5-ba7f-568be2b6a69a/inputs` -> 200, `total=7`, no zero-value timestamps.
- Local frontend: `http://127.0.0.1:5182/pipeline/batch/3224bbdd-121a-40a5-ba7f-568be2b6a69a?verify=local-run-tree-health`; Chrome DevTools network requests completed 200, console had no warn/error.

## Push Strategy
- [x] Keep work on local `dev` until this hardening slice is implemented and verified.
- [ ] Push to `dev` after deploy verification succeeds.
