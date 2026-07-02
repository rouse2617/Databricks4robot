# Tasks - CYB-3007

## Context files

- `Frontend/src/pages/WorkflowExecutionList.tsx`
- `Frontend/src/pages/WorkflowExecutionList.test.tsx`
- `Frontend/src/pages/PipelinePage.tsx`
- `Frontend/src/pages/PipelinePage.test.tsx`
- `Frontend/src/lib/pipelineNavigation.ts`
- `Frontend/src/components/pipeline/DeployPanel.tsx`
- `Frontend/src/api/runApi.ts`
- `Frontend/src/api/workflowApi.ts`

## OpenSpec

- [x] [openspec] Create proposal, design, tasks, decisions, context files, and runtime-os spec delta.
- [x] [openspec] Continue without stopping for another checkpoint per user instruction.

## Implementation

- [x] [Frontend] Remove `listWorkflows` from Execution Hub list refresh.
- [x] [Frontend] Remove template inference based on live workflow names from the list path.
- [x] [Frontend] Project execution table rows only from Run ledger rows.
- [x] [Frontend] Keep local filters based on available Run fields.
- [x] [Frontend] Keep runtime workflow APIs outside the product list path.
- [x] [Frontend] Add `/runs` as the stable run-centric execution list route.
- [x] [Frontend] Move deploy, batch, and detail return navigation to `/runs` when a Run id is available.
- [x] [Frontend] Move asset lineage Run navigation to `/runs/{runId}` when lineage exposes a deployment id.
- [x] [Frontend] Add execution table sorters for duration, estimated total cost, created time, and finished time.
- [x] [Frontend] Add regression coverage for sortable execution metric columns.
- [x] [backend] Add Run summary `q` search and template-name matching.
- [x] [backend] Include persisted summary `totalEstimatedCost` and `templateName` in Run list rows.
- [x] [Frontend] Pass the execution search box to the Run list API and render template names.
- [x] [api] Sync OpenAPI and API guide for Run summary cost/template search.
- [x] [scripts] Extend Run API smoke coverage for summary query/cost fields.
- [x] [backend] Reconcile bounded anomalous terminal Run rows from the watcher without adding Argo calls to default `/runs` list reads.
- [x] [backend] Sync repaired batch child Run statuses back to linked backfill items.

## Verification

- [x] [Frontend] `cd Frontend && npx biome check src/pages/WorkflowExecutionList.tsx src/pages/WorkflowExecutionList.test.tsx`.
- [x] [Frontend] `cd Frontend && npm run test -- src/pages/WorkflowExecutionList.test.tsx --run`.
- [x] [backend] `cd backend && go test ./internal/handlers/pipeline ./internal/postgres/...`.
- [x] [backend] `cd backend && go test ./internal/usecase/pipeline`.
- [x] [backend] `cd backend && go test ./internal/usecase/pipeline ./internal/usecase/backfill`.
- [x] [Frontend] `cd Frontend && npx biome check src/pages/WorkflowExecutionList.tsx src/pages/WorkflowExecutionList.test.tsx src/api/runApi.ts src/api/runApi.test.ts`.
- [x] [Frontend] `cd Frontend && npm run test -- src/pages/WorkflowExecutionList.test.tsx src/api/runApi.test.ts --run`.
- [x] [Frontend] `cd Frontend && npx biome check src/pages/PipelinePage.tsx src/pages/PipelinePage.test.tsx src/lib/pipelineNavigation.ts src/components/pipeline/DeployPanel.tsx`.
- [x] [Frontend] `cd Frontend && npm run test -- src/pages/PipelinePage.test.tsx src/lib/pipelineNavigation.test.ts src/components/pipeline/DeployPanel.test.tsx --run`.
- [x] [Frontend] Include this page in the final full frontend build/test pass: `cd Frontend && npm run test -- --run` and `cd Frontend && npm run build`.
- [x] [Local UI] Verified local frontend at `http://127.0.0.1:5181/runs` and `http://127.0.0.1:5181/runs/9981892a-1873-4715-8f64-e785009ed5db`.
- [ ] [repo] Non-Terraform pre-commit checks before final commit when the full runtime goal is ready.

## Deploy Verification

- [x] Deployed backend dev image `cyber-databrew-backend:81cfddd-runtimeos-20260620100915` to revision `cyber-databrew-backend-dev-00966-564`.
- [x] Deployed backend dev image `cyber-databrew-backend:1792c0a9-watcher-anomaly-20260623193359` to revision `cyber-databrew-backend-dev-01048-b69`.
- [x] Backend dev smoke after watcher anomaly deploy: `/readyz` healthy; `/api/v1/runs/watcher/status` returned `healthy=true, stale=false, lastSyncedRunCount=58`; `/api/v1/runs?view=summary&excludeBatch=true&page=1&pageSize=2` returned 200 with 2 items.
- [x] Deployed frontend dev image `cyber-databrew-frontend:81cfddd-runtimeos-20260620101541` to revision `cyber-databrew-frontend-dev-00398-fk7`.
- [x] Published Cloudflare Worker static assets for `https://cyber-databrew-dev.cyberorigin.ai/`, version `18e8d1c0-8174-4732-b6c0-df57c57414b9`.
- [x] Used Chrome DevTools MCP to verify deployed `/runs` list loads from the public dev domain without runtime console errors.
- [x] Used Chrome DevTools MCP to verify deployed run detail logs, Pod diagnostics, and resource monitoring requests return 200.
- [x] Captured local deployed verification screenshots (kept in the workspace; `openspec/changes/**/*.png` is gitignored):
  - `openspec/changes/CYB-3007-run-centric-execution-list/deploy-verify-public-runs.png`
  - `openspec/changes/CYB-3007-run-centric-execution-list/deploy-verify-public-logs.png`
  - `openspec/changes/CYB-3007-run-centric-execution-list/deploy-verify-public-pod.png`

## Push Strategy

- [ ] Do not open or push a PR for this slice alone.
- [ ] Keep work on local `dev`; push once only after the full Run as Kernel goal is complete and verified.
