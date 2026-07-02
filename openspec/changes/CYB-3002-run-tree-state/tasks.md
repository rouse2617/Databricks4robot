# Tasks — CYB-3002

## Context files
- `backend/internal/runtimeos/run/service.go`
- `backend/internal/runtimeos/state/state_machine.go`
- `backend/internal/usecase/pipeline/usecase.go`
- `backend/internal/usecase/pipeline/batch_subtask.go`
- `backend/internal/usecase/backfill/usecase.go`
- `backend/internal/models/pipeline.go`
- `backend/internal/repository/pipeline_repository.go`
- `backend/internal/handlers/pipeline/handler.go`
- `backend/routes/routes.go`
- `Frontend/src/api/runApi.ts`
- `Frontend/src/pages/BatchJobDetailPage.tsx`
- `Frontend/src/pages/WorkflowExecutionList.tsx`
- `api/openapi.yaml`
- `docs/review/api-guide.md`
- `scripts/smoke-runs-dev.sh`
- `sdk/tests/unit/test_managers.py`

## OpenSpec
- [x] [openspec] Create proposal, design, tasks, decisions, context files, and runtime-os spec delta for Run Tree phase two.
- [x] [openspec] Record user approval to proceed past the OpenSpec checkpoint before editing runtime code.

## Implementation
- [x] [backend] Add `runtimeos/state` status normalization and child aggregation helper with table-driven tests.
- [x] [backend] Add `RunRelation` and `RunChildSummary` projection models.
- [x] [backend] Update `ListRunChildren` to return child Runs, relation projections, and aggregate summary without requiring live Argo workflows.
- [x] [backend] Add normalized Run diagnostic projection fields (`failureReason`, `blockingReason`, `blockingMessage`) from existing status/message data.
- [x] [backend] Add `topFailureReasons` aggregation to child Run summaries.
- [x] [backend] Project generic failure reasons when failed Runs have no top-level message and infer Run diagnostics from failed node messages.
- [x] [backend] Add `hasFailures` and `hasBlocking` child summary flags so active batches can show already-failed or blocked work.
- [x] [backend] Ensure new backfill/batch jobs can materialize a parent Run record that `/runs/:id` can inspect.
- [x] [backend] Preserve legacy backfill, deployment, workflow, and pipeline-run endpoint behavior.
- [x] [Frontend] Type enriched `RunChildrenResponse` in `runApi`.
- [x] [Frontend] Show Run Tree summary and child Run links on Batch Detail using `runApi.getRunChildren` or equivalent Run API calls.
- [x] [Frontend] Surface normalized blocking/failure reason tags in Execution Hub and top child reasons in Batch Detail.
- [x] [Frontend] Keep `/runs/:id` Run-ledger centered when the runtime workflow is missing or not yet submitted.
- [x] [Frontend] Show Batch Run Tree health tags when an active batch already has failures or blocking children.

## API Contract Sync
- [x] [api] Update `api/openapi.yaml` for enriched `RunChildList`, `RunRelation`, and `RunChildSummary`.
- [x] [api] Update `api/openapi.yaml` for Run diagnostic reason fields and `RunBlockingReason`.
- [x] [api] Update `api/openapi.yaml`, SDK generated models, smoke script, and frontend types for `RunChildSummary.hasFailures/hasBlocking`.
- [x] [docs] Update `docs/review/api-guide.md` to document Batch as parent Run plus child Runs.
- [x] [scripts] Extend `scripts/smoke-runs-dev.sh` to verify children summary fields on a known or synthetic Run Tree when available.
- [x] [openspec] Keep behavior delta aligned with implemented scope.
- [x] [Frontend] Keep `Frontend/src/api/runApi.ts` response types aligned with OpenAPI.

## Verification
- [x] [backend] `cd backend && go test ./internal/runtimeos/state ./internal/usecase/pipeline ./internal/usecase/backfill ./internal/handlers/pipeline`.
- [x] [backend] `cd backend && go test ./...`.
- [x] [Frontend] `cd Frontend && npm run test -- src/api/runApi.test.ts src/pages/BatchJobDetailPage.test.tsx --run`.
- [x] [Frontend] `cd Frontend && npm run test -- src/pages/WorkflowDetailPage.test.tsx --run`.
- [x] [Frontend] `cd Frontend && npm run build`.
- [x] [SDK] `cd sdk && uv run pytest tests/unit/ -q`.
- [x] [repo] `git diff --check`.
- [x] [Frontend] `cd Frontend && npx biome check src/api/runApi.ts src/api/runApi.test.ts src/pages/BatchJobDetailPage.tsx src/pages/BatchJobDetailPage.test.tsx`.
- [x] [Frontend] Wording-only label update: `cd Frontend && npm run test -- src/pages/BatchJobDetailPage.test.tsx --run`.
- [x] [Frontend] Wording-only label update: `cd Frontend && npx biome check src/pages/BatchJobDetailPage.tsx src/pages/BatchJobDetailPage.test.tsx`.
- [x] [Frontend] Wording-only label update: `cd Frontend && npm run build`.
- [ ] [Frontend] `cd Frontend && npm run lint` (currently blocked by pre-existing unrelated Biome findings outside this diff).
- [x] [repo] `SKIP=terraform_fmt,terraform_validate,terraform_tflint uvx --python 3.12 pre-commit run --all-files`.

## Deploy Verification
- [x] Deploy backend dev with SHA image and record revision.
- [x] Smoke `/api/v1/runs/:id/children` for child Runs, relation projections, and aggregate summary.
- [x] If frontend files changed, deploy frontend dev and verify Batch Detail/Run Tree with Chrome DevTools MCP.
- [x] Update public dev Cloudflare Worker/Assets entry and verify `https://cyber-databrew-dev.cyberorigin.ai` loads the latest frontend chunk.

### Deploy record
| Service | Image tag / version | Revision | URL |
|---------|---------------------|----------|-----|
| backend-dev | `cyber-databrew-backend:054576d-run-diag2-20260620185222` | `cyber-databrew-backend-dev-00968-nn8` | `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app` |
| frontend-dev | `cyber-databrew-frontend:054576d-run-diag2-20260620185834` | `cyber-databrew-frontend-dev-00399-kn8` | `https://cyber-databrew-frontend-dev-wtttm6suaq-uc.a.run.app` |
| public-dev-worker | `wrangler deploy --assets /tmp/cyb3002-site-publish.QG1lPk` | Worker version `3e68c25a-d193-4f03-8480-d3d6a91047ac` | `https://cyber-databrew-dev.cyberorigin.ai` |
| frontend-dev | `cyber-databrew-frontend:8522248-run-ledger-fallback2-20260620194756` | `cyber-databrew-frontend-dev-00401-sx4` | `https://cyber-databrew-frontend-dev-wtttm6suaq-uc.a.run.app` |
| public-dev-worker | `wrangler deploy --assets /tmp/cyb3002-ledger-site-publish.yznz6p` | Worker version `d4c5901a-b602-403b-af9b-85bd98ed304f` | `https://cyber-databrew-dev.cyberorigin.ai` |
| backend-dev | `cyber-databrew-backend:c43d921-run-diag-fallback-20260620201905` | `cyber-databrew-backend-dev-00970-2m9` | `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app` |
| local-vite | `v0.1.1 (local)` | `http://127.0.0.1:5182` | Cloud Run backend `cyber-databrew-backend-dev-00970-2m9` |

### Deploy verification evidence
- Backend smoke: `source scripts/dev-backend-env.sh && bash scripts/smoke-runs-dev.sh` -> `23 passed, 0 failed`.
- API spot-check: `/api/v1/runs?view=summary&page=1&pageSize=50` returns normalized `failureReason` values including `resource_incompatible`, `runtime_missing`, and `image_startup`.
- Synthetic batch job: `02972429-30df-49dd-9ba4-5bdf50069549`; child Run: `9981892a-1873-4715-8f64-e785009ed5db`.
- API smoke: `/api/v1/runs/02972429-30df-49dd-9ba4-5bdf50069549/children` returned `total=1`, `summary.aggregateStatus=Running/Succeeded`, and `relations[0].relationType=batch_child`.
- Chrome MCP screenshot: `deploy-verify-batch-run-tree.png` shows public dev Batch Detail with `运行树`, `关系 1`, and child Run link.
- Chrome MCP screenshot: `deploy-verify-child-run-pod-monitoring.png` shows child Run logs, Pod diagnostics, resource snapshot, and terminal disabled because the Pod is completed.
- Chrome MCP network checks: public dev Batch Detail requested `/api/v1/runs/02972429-30df-49dd-9ba4-5bdf50069549/children` with 200; child Run logs, pod diagnostics, and resources endpoints all returned 200.
- Chrome MCP console checks: no runtime console errors on Batch Detail; child Run only reports an existing accessibility issue for one form field missing `id`/`name`.
- Performance trace: `perf-batch-run-tree-public-dev.trace.json.json.gz`; LCP 1365 ms, CLS 0.01, render-blocking estimated savings 0 ms.
- After user feedback, visible UI copy changed from `运行树`/`关系` to `批次运行`/`子运行`. This wording-only update was verified locally at `http://127.0.0.1:5178/pipeline/batch/02972429-30df-49dd-9ba4-5bdf50069549?verify=cyb3002-local-label` against the dev backend; console had no errors and `/api/v1/runs/.../children` returned 200.
- Final Chrome MCP public dev `/runs`: `deploy-verify-runs-diagnostics-final.png` shows reason tags such as `Runtime 不可用`, `镜像启动`, and `调度失败`; network `/api/v1/runs?view=summary&excludeBatch=true&page=1&pageSize=20` returned 200.
- Final Chrome MCP public dev Batch Detail: `deploy-verify-batch-children-final.png` shows `批次运行`, `子运行 1`, and child Run link; `/api/v1/runs/02972429-30df-49dd-9ba4-5bdf50069549/children`, `/api/v1/backfill/...`, and child summary requests returned 200; console had no messages.
- Final Chrome MCP frontend Cloud Run: `deploy-verify-cloudrun-frontend-final.png` shows version `v054576d-run-diag2-20260620185834 (dev#054576d)`; console had no messages and app data requests returned 200 after initial dev auth.
- Final Chrome MCP local Vite: `deploy-verify-local-runs-final.png` at `http://127.0.0.1:5181/runs?verify=cyb3002-final` shows the same normalized reason tags; `/api/v1/runs?view=summary&excludeBatch=true&page=1&pageSize=20` returned 200.
- Chrome MCP public dev Run ledger fallback: `deploy-verify-run-ledger-fallback-final.png` at `/runs/e0fa8796-5564-4ddb-a3a1-4d7d58c7c899?verify=cyb3002-ledger-fallback-final` shows `底层 Runtime 已不可用，正在展示 DataBrew 历史账本`, Run inputs, `运行上下文`, `video-proc-dev`, saved config inputs, and diagnostic tag `Runtime 不可用`.
- Chrome MCP network checks for the fallback page: `/api/v1/runs/:id`, `/events`, `/asset-nodes`, `/cost-summary`, `/inputs`, `/outputs`, and `/runtime` returned 200; `/api/v1/workflows/cyb2224-video-proc-five-step-smoke-disk20-20260618173304-88fa0a` returned the expected 404 that triggered ledger fallback.
- Chrome MCP console check for the fallback page: one expected resource error from the workflow 404; no React/runtime exceptions.
- Backend smoke after fallback deploy: `source scripts/dev-backend-env.sh && bash scripts/smoke-runs-dev.sh` -> `23 passed, 0 failed`.
- API spot-check after fallback deploy: `/api/v1/runs/3224bbdd-121a-40a5-ba7f-568be2b6a69a/children` returned 200 for a legacy batch without a parent Run projection, `summary.aggregateStatus=Running`, `summary.hasFailures=true`, and top reason `runtime_missing`.
- Chrome MCP local Vite Run detail: `local-run-detail-20260620.png` at `http://127.0.0.1:5182/runs/a573704b-6644-41f1-bda4-f3b6bd797ec0?verify=local-run-diagnostics`; `/api/v1/runs/:id`, `/events`, `/asset-nodes`, `/cost-summary`, `/inputs`, `/outputs`, `/runtime`, and workflow requests returned 200.
- Chrome MCP local Vite Batch Detail: `local-batch-run-tree-health-20260620.png` at `http://127.0.0.1:5182/pipeline/batch/3224bbdd-121a-40a5-ba7f-568be2b6a69a?verify=local-run-tree-health` shows `批次运行`, `已有失败`, `主要阻塞 / 失败原因`, and `Runtime 不可用`; all batch and Run Tree requests returned 200 and console had no errors beyond Vite/React dev info.

## PR
- [ ] PR template includes Linear placeholder/real CYB, OpenSpec change-id, test evidence, deploy evidence, and any local environment limitations.
