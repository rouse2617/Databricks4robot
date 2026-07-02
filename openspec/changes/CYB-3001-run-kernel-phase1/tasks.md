# Tasks — CYB-3001

## Context files
- `backend/routes/routes.go`
- `backend/internal/handlers/pipeline/handler.go`
- `backend/internal/usecase/pipeline/usecase.go`
- `backend/internal/models/pipeline.go`
- `backend/internal/repository/pipeline_repository.go`
- `sdk/src/cyber_databrew_sdk/managers/runs.py`
- `sdk/src/cyber_databrew_sdk/config/endpoints.py`
- `Frontend/src/api/pipelineApi.ts`
- `Frontend/src/api/runApi.ts`
- `Frontend/src/pages/WorkflowExecutionList.tsx`
- `Frontend/src/pages/WorkflowDetailPage.tsx`
- `Frontend/src/pages/useWorkflowDetail.ts`
- `Frontend/src/App.tsx`
- `api/openapi.yaml`
- `docs/review/api-guide.md`

## OpenSpec
- [x] [openspec] Create proposal, design, tasks, decisions, context files, and spec delta for Phase 1.
- [x] [openspec] Record the explicit user autonomy instruction that allows continuing past the checkpoint for this iteration.

## Implementation
- [x] [backend] Register `/api/v1/runs` collection, item, by-workflow, subresource, and operation routes while keeping `/pipeline-runs`, `/deployments`, and `/workflows`.
- [x] [backend] Add run-centric handler methods for nodes, inputs, outputs, children, and runtime views.
- [x] [backend] Add usecase projection helpers for Run nodes, inputs, outputs, children, and runtime refs.
- [x] [backend] Add runtime retry/resubmit/suspend/resume/terminate methods for the new `/runs` API; keep legacy `/pipeline-runs/:id/retry` full-rerun behavior.
- [x] [Frontend] Add `Frontend/src/api/runApi.ts` with run-centric methods and tests.
- [x] [Frontend] Add `/runs` and `/runs/:runId` routes; route `/runs` to the execution hub and `/runs/:runId` to the current inspector by run id.
- [x] [Frontend] Start replacing new execution-detail API calls with `runApi` without breaking legacy workflowName links.
- [x] [SDK] Add `client.runs` manager for product Run API without changing legacy `client.pipelines` methods.
- [x] [Frontend] Switch Execution Hub primary list data from `pipelineApi.listPipelineRuns` to `runApi.listRuns`.
- [x] [Frontend] Keep live Argo workflow list as optional debug/filter enrichment only; do not merge live-only workflows into product execution rows.
- [x] [Frontend] Resolve legacy workflowName detail links through `runApi.getRunByWorkflowName` and use run subresources after resolution.
- [x] [Frontend] Route Run Inspector delete/retry/resubmit/stop/suspend/resume/terminate actions through `runApi` when a Run exists, preserving workflow debug fallback for external workflows.
- [x] [Frontend] Route single-template execution creation through `runApi.createRunByTemplate` instead of legacy deploy/pipeline-run helpers.
- [x] [backend] Introduce `internal/runtimeos/run.Service` as the Run Kernel facade over the existing pipeline usecase and route product Run endpoints through it.
- [x] [backend] Introduce `internal/runtimeos/adapter.RuntimeAdapter` and the first Argo adapter implementation over the existing Argo workflow client.
- [x] [backend] Make Run stop operations write requested, succeeded, and failed RunEvent ledger entries.
- [x] [backend] Make Run retry/resubmit creation failures write failed RunEvent ledger entries on the source Run.
- [x] [backend] Omit Run runtime `debugUrl` when the Run has no workflow name.

## API Contract Sync
- [x] [api] Update `api/openapi.yaml` for `/api/v1/runs` paths and schemas.
- [x] [docs] Update `docs/review/api-guide.md` with Run = product execution, Workflow = runtime debug, Deployment = legacy compatibility.
- [x] [scripts] Add or update smoke coverage for run API happy path and at least one error path.
- [x] [openspec] Keep behavior delta aligned with implemented scope.

## Verification
- [x] [backend] `cd backend && go test ./internal/handlers/pipeline ./internal/usecase/pipeline`.
- [x] [backend] `cd backend && go test ./internal/handlers/pipeline ./internal/usecase/pipeline ./routes`.
- [x] [backend] `cd backend && go test ./internal/runtimeos/run ./internal/handlers/pipeline ./internal/usecase/pipeline ./routes`.
- [x] [backend] `cd backend && go test ./internal/runtimeos/...`.
- [x] [backend] `cd backend && go test ./internal/runtimeos/... ./internal/handlers/pipeline ./internal/usecase/pipeline ./routes`.
- [x] [backend] `cd backend && go test ./internal/usecase/pipeline ./internal/handlers/pipeline`.
- [x] [backend] `cd backend && go test ./internal/usecase/pipeline ./internal/handlers/pipeline ./routes && go test ./internal/runtimeos/... ./internal/handlers/pipeline ./internal/usecase/pipeline ./routes && go test ./...`.
- [x] [backend] `cd backend && go test ./internal/usecase/pipeline ./internal/handlers/pipeline ./routes && go test ./...` after the runtime debug URL guard.
- [x] [backend] `cd backend && go test ./...` before PR/deploy unless environment blocks it.
- [x] [Frontend] `cd Frontend && npm run test -- --run src/api/runApi.test.ts src/pages/WorkflowDetailPage.test.tsx src/pages/useWorkflowDetail.test.tsx`.
- [x] [Frontend] `cd Frontend && npm run test -- src/api/runApi.test.ts src/pages/WorkflowExecutionList.test.tsx src/pages/WorkflowDetailPage.test.tsx src/pages/useWorkflowDetail.test.tsx` (`37 passed`).
- [x] [Frontend] `cd Frontend && npm run test -- src/api/deployPipelineRun.test.ts src/api/runApi.test.ts`.
- [x] [Frontend] `cd Frontend && npx biome check src/api/deployPipelineRun.ts src/api/deployPipelineRun.test.ts src/api/runApi.ts src/api/runApi.test.ts`.
- [x] [Frontend] `cd Frontend && npm run build`.
- [x] [Frontend] `cd Frontend && npx biome check src/api/runApi.ts src/api/runApi.test.ts src/App.tsx src/pages/WorkflowDetailPage.tsx src/pages/WorkflowDetailPage.test.tsx`.
- [x] [Frontend] `cd Frontend && npx biome check src/api/runApi.ts src/pages/WorkflowExecutionList.tsx src/pages/WorkflowExecutionList.test.tsx src/pages/WorkflowDetailPage.tsx src/pages/WorkflowDetailPage.test.tsx src/pages/useWorkflowDetail.ts src/pages/useWorkflowDetail.test.tsx`.
- [ ] [Frontend] `cd Frontend && npm run lint` (blocked by existing unrelated Biome issues in `src/index.css`, `src/styles/pipeline.css`, and other untouched files).
- [ ] [Frontend] `cd Frontend && npm run test -- --run` (blocked by existing unrelated tests expecting old English labels/old placeholders in Assets/Events pages).
- [x] [SDK] `cd sdk && uv run ruff check src/`.
- [x] [SDK] `cd sdk && uv run pytest tests/unit/`.
- [x] [repo] `git diff --check`.

## Deploy Verification
- [x] Deploy backend dev with SHA image and record revision.
- [x] Smoke `/api/v1/runs`, `/api/v1/runs/:id`, `/api/v1/runs/by-workflow/:workflowName`, `/api/v1/runs/:id/events`, `/inputs`, `/outputs`, `/children`, `/runtime`.
- [x] If frontend files changed, deploy frontend dev and verify `/runs` and `/runs/:runId` with Chrome DevTools MCP.
- [x] Redeploy backend dev after the RuntimeAdapter slice and repeat Run API smoke before committing.

### Deploy record
| Service | Image tag / version | Revision | URL |
|---------|---------------------|----------|-----|
| backend-dev | `cyber-databrew-backend:b94beb4-cyb3001-runapi2-20260619175638` | `cyber-databrew-backend-dev-00949-26l` | `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app` |
| frontend-dev | `cyber-databrew-frontend:b7689e2-cyb3001-runhub-stable-key-20260620025229` | `cyber-databrew-frontend-dev-00395-4wl` | `https://cyber-databrew-frontend-dev-wtttm6suaq-uc.a.run.app` |
| public dev Worker | `cyber-databrew-dev` Worker Assets from `Frontend/dist` | version `3a46dde6-feea-4f46-b1d4-7de2eccd75c4` | `https://cyber-databrew-dev.cyberorigin.ai` |
| backend-dev | `cyber-databrew-backend:b98a76c-cyb3001-runservice2-20260620032310` | `cyber-databrew-backend-dev-00954-gzm` | `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app` |
| frontend-dev | `cyber-databrew-frontend:b98a76c-cyb3001-runservice2-20260620032324` | `cyber-databrew-frontend-dev-00396-m2v` | `https://cyber-databrew-frontend-dev-wtttm6suaq-uc.a.run.app` |
| public dev Worker | `cyber-databrew-dev` Worker Assets from `Frontend/dist` | version `2fb301fe-4a23-4676-8c86-36229b98395d` | `https://cyber-databrew-dev.cyberorigin.ai` |
| backend-dev | `cyber-databrew-backend:b98a76c-cyb3001-runtimeadapter-20260620033735` | `cyber-databrew-backend-dev-00955-pfd` | `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app` |
| backend-dev | `cyber-databrew-backend:a1616d0-cyb3001-stop-events-20260620035453` | `cyber-databrew-backend-dev-00957-jk4` | `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app` |
| backend-dev | `cyber-databrew-backend:a1616d0-cyb3001-run-events2-20260620040154` | `cyber-databrew-backend-dev-00958-jr9` | `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app` |
| backend-dev | `cyber-databrew-backend:6b2fb4c-cyb3001-debugurl-20260619201744` | `cyber-databrew-backend-dev-00960-k8j` | `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app` |

### Deploy smoke
- [x] `bash scripts/smoke-runs-dev.sh` against Cloud Run backend dev: `16 passed, 0 failed`.
- [x] Public `/runs` Chrome MCP: `deploy-verify-public-runs-stable-key.snapshot.txt`, `deploy-verify-public-runs-stable-key.png`; network loaded `/api/v1/runs?view=summary&excludeBatch=true&page=1&pageSize=20` and `/api/v1/pipelines?page_size=200`; no legacy `/api/v1/pipeline-runs` request.
- [x] Public `/runs/:runId` Chrome MCP: `deploy-verify-public-run-detail-stable-key.snapshot.txt`, `deploy-verify-public-run-detail-stable-key.png`; network loaded `/api/v1/runs/:id`, `/events`, `/asset-nodes`, `/cost-summary`, and `/api/v1/workflows/cyb2224-terminal-final-231444-4c5a9d`; no legacy `/api/v1/workflows/<runId>` request.
- [x] Public legacy `/pipeline/executions/:workflowName` Chrome MCP: `deploy-verify-public-legacy-workflow-link-stable-key.snapshot.txt`, `deploy-verify-public-legacy-workflow-link-stable-key.png`; network loaded `/api/v1/workflows/:workflowName`, `/api/v1/runs/by-workflow/:workflowName`, then Run subresources by stable run id.
- [x] Cloud Run frontend `/runs` Chrome MCP: `deploy-verify-cloudrun-runs-stable-key.snapshot.txt`, `deploy-verify-cloudrun-runs-stable-key.png`; route reached execution list with the final image version visible and loaded `/api/v1/runs?view=summary&excludeBatch=true&page=1&pageSize=20`.
- [x] Cloud Run frontend `/runs/:runId` Chrome MCP: `deploy-verify-cloudrun-run-detail-stable-key.snapshot.txt`, `deploy-verify-cloudrun-run-detail-stable-key.png`; network loaded Run subresources and `/api/v1/workflows/cyb2224-terminal-final-231444-4c5a9d`, with no `/api/v1/workflows/<runId>` request.
- [x] Cloud Run backend smoke after `00954-gzm`: `/readyz` healthy; `/api/v1/runs?view=summary&excludeBatch=true&page=1&pageSize=2` returned `total=27`; `/api/v1/runs/watcher/status` returned `healthy=true`; nonexistent `/api/v1/runs/template/not-a-template-for-cyb3001` returned `404 template not found`.
- [x] Public Worker deploy `2fb301fe-4a23-4676-8c86-36229b98395d`: `Frontend/dist` contains `/runs/template/${templateId}` in the non-map JS bundle.
- [x] Public execution list after Worker deploy: `deploy-verify-public-worker-runservice-runs.png`; network loaded `/api/v1/runs?view=summary&excludeBatch=true&page=1&pageSize=20` and no `/api/v1/pipeline-runs` product list request; console had no JS error/warn.
- [x] Public run detail diagnostics after backend/frontend deploy: `deploy-verify-public-runservice-detail-diagnostics.snapshot.txt`, `deploy-verify-public-runservice-detail-diagnostics.png`; network loaded `/api/v1/runs/:id`, Run subresources, `/workflows/:workflowName/logs`, `/logs/stream`, `/nodes/:nodeId/pod`, and `/nodes/:nodeId/resources` with 200 responses. Logs and bounded live SSE rendered; Pod diagnostics and resource snapshot rendered; terminal showed clear disabled state for completed Pod.
- [x] Cloud Run backend smoke after RuntimeAdapter deploy `00955-pfd`: `/readyz` healthy; `/api/v1/runs?view=summary&excludeBatch=true&page=1&pageSize=2` returned `total=27`; `/api/v1/runs/watcher/status` returned `healthy=true, stale=false`; nonexistent `/api/v1/runs/template/not-a-template-for-cyb3001-runtimeadapter` returned `404 ASSET_NOT_FOUND`.
- [x] Cloud Run backend smoke after stop-event deploy `00957-jk4`: `/readyz` healthy; `/api/v1/runs?view=summary&excludeBatch=true&page=1&pageSize=2` returned `total=27`; `/api/v1/runs/watcher/status` returned `healthy=true, stale=false`; nonexistent `/api/v1/runs/template/not-a-template-for-cyb3001-stop-events` returned `404 ASSET_NOT_FOUND`; nonexistent `/api/v1/runs/not-a-run-for-cyb3001-stop-events/stop` returned `404 ASSET_NOT_FOUND`.
- [x] Cloud Run backend smoke after retry/resubmit failed-event deploy `00958-jr9`: `/readyz` healthy; `/api/v1/runs?view=summary&excludeBatch=true&page=1&pageSize=2` returned `total=27`; `/api/v1/runs/watcher/status` returned `healthy=true, stale=false`; nonexistent `/api/v1/runs/template/not-a-template-for-cyb3001-run-events2` returned `404 ASSET_NOT_FOUND`; nonexistent `/api/v1/runs/not-a-run-for-cyb3001-run-events2/stop` returned `404 ASSET_NOT_FOUND`.
- [x] Cloud Run backend smoke after debug URL guard deploy `00960-k8j`: `/readyz` healthy; `/api/v1/runs?view=summary&excludeBatch=true&page=1&pageSize=2` returned `total=27`; `/api/v1/runs/watcher/status` returned `healthy=true, stale=false`; nonexistent template POST and stop returned `404 ASSET_NOT_FOUND`; `/api/v1/runs/:id/runtime` returned workflow runtime debug URL for an existing workflow-backed Run.

## PR
- [ ] PR template includes Linear placeholder/real CYB, OpenSpec change-id, test evidence, deploy evidence, and the Linear credential limitation if still unresolved.
