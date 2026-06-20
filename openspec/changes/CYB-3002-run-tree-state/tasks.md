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
- [x] [backend] Ensure new backfill/batch jobs can materialize a parent Run record that `/runs/:id` can inspect.
- [x] [backend] Preserve legacy backfill, deployment, workflow, and pipeline-run endpoint behavior.
- [x] [Frontend] Type enriched `RunChildrenResponse` in `runApi`.
- [x] [Frontend] Show Run Tree summary and child Run links on Batch Detail using `runApi.getRunChildren` or equivalent Run API calls.

## API Contract Sync
- [x] [api] Update `api/openapi.yaml` for enriched `RunChildList`, `RunRelation`, and `RunChildSummary`.
- [x] [docs] Update `docs/review/api-guide.md` to document Batch as parent Run plus child Runs.
- [x] [scripts] Extend `scripts/smoke-runs-dev.sh` to verify children summary fields on a known or synthetic Run Tree when available.
- [x] [openspec] Keep behavior delta aligned with implemented scope.
- [x] [Frontend] Keep `Frontend/src/api/runApi.ts` response types aligned with OpenAPI.

## Verification
- [x] [backend] `cd backend && go test ./internal/runtimeos/state ./internal/usecase/pipeline ./internal/usecase/backfill ./internal/handlers/pipeline`.
- [x] [backend] `cd backend && go test ./...`.
- [x] [Frontend] `cd Frontend && npm run test -- src/api/runApi.test.ts src/pages/BatchJobDetailPage.test.tsx --run`.
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
| backend-dev | `cyber-databrew-backend:4d51ec2-cyb3002-runtree-20260620154925` | `cyber-databrew-backend-dev-00963-wmj` | `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app` |
| frontend-dev | `cyber-databrew-frontend:4d51ec2-cyb3002-runtree-20260620154925` | `cyber-databrew-frontend-dev-00397-rbd` | `https://cyber-databrew-frontend-dev-wtttm6suaq-uc.a.run.app` |
| public-dev-worker | `wrangler deploy --assets /tmp/cyb3002-site-publish.*` | Worker version `594741cd-eb4b-4ee4-b2f8-231e95e23fbd` | `https://cyber-databrew-dev.cyberorigin.ai` |

### Deploy verification evidence
- Backend smoke: `source scripts/dev-backend-env.sh && bash scripts/smoke-runs-dev.sh` -> `16 passed, 0 failed`.
- Synthetic batch job: `02972429-30df-49dd-9ba4-5bdf50069549`; child Run: `9981892a-1873-4715-8f64-e785009ed5db`.
- API smoke: `/api/v1/runs/02972429-30df-49dd-9ba4-5bdf50069549/children` returned `total=1`, `summary.aggregateStatus=Running/Succeeded`, and `relations[0].relationType=batch_child`.
- Chrome MCP screenshot: `deploy-verify-batch-run-tree.png` shows public dev Batch Detail with `运行树`, `关系 1`, and child Run link.
- Chrome MCP screenshot: `deploy-verify-child-run-pod-monitoring.png` shows child Run logs, Pod diagnostics, resource snapshot, and terminal disabled because the Pod is completed.
- Chrome MCP network checks: public dev Batch Detail requested `/api/v1/runs/02972429-30df-49dd-9ba4-5bdf50069549/children` with 200; child Run logs, pod diagnostics, and resources endpoints all returned 200.
- Chrome MCP console checks: no runtime console errors on Batch Detail; child Run only reports an existing accessibility issue for one form field missing `id`/`name`.
- Performance trace: `perf-batch-run-tree-public-dev.trace.json.json.gz`; LCP 1365 ms, CLS 0.01, render-blocking estimated savings 0 ms.
- After user feedback, visible UI copy changed from `运行树`/`关系` to `批次运行`/`子运行`. This wording-only update was verified locally at `http://127.0.0.1:5178/pipeline/batch/02972429-30df-49dd-9ba4-5bdf50069549?verify=cyb3002-local-label` against the dev backend; console had no errors and `/api/v1/runs/.../children` returned 200.

## PR
- [ ] PR template includes Linear placeholder/real CYB, OpenSpec change-id, test evidence, deploy evidence, and any local environment limitations.
