# Tasks — CYB-2007

## Context files
- `backend/internal/usecase/backfill/usecase.go` — batch creation, scheduling, reconciliation, progress, node summary
- `backend/internal/usecase/backfill/usecase_test.go` — existing backfill unit tests and mocks
- `backend/internal/postgres/backfill_repo.go` — item aggregation and node coverage queries
- `backend/internal/postgres/pipeline_repo.go` — batch child execution list filtering and totals
- `backend/internal/usecase/pipeline/batch_subtask.go` — batch child run upsert behavior
- `Frontend/src/pages/BatchJobDetailPage.tsx` — deployed UI surface used for regression verification
- `Frontend/src/pages/WorkflowExecutionList.tsx` — batch child list UI and filters

## Implementation
- [x] [backend] Add a failing unit test proving `CreateBackfill` does not save duplicate backfill items before and during scheduling.
- [x] [backend] Change batch materialization so scheduling reuses the already-created items instead of inserting a second copy.
- [x] [backend] Add or update tests proving `syncJobProgress` sets a fully completed batch to `completed`.
- [x] [backend] Add or update tests proving node summary uses logical batch item totals and marks complete when each logical item has node rows.
- [x] [backend] Ensure `pipeline-runs?view=summary&batchJobId=...` represents one visible child run per logical asset for the batch detail default view.
- [x] [backend] Keep rerun behavior explicit: reruns may create new attempts, but default batch detail progress remains keyed to current logical item state.

## API contract sync
No new endpoint, parameter, response field, or status code is planned. This change tightens existing response semantics only.

## Local verification
- [x] `cd backend && go test ./internal/usecase/backfill/...`
- [x] `cd backend && go test ./internal/postgres/...`
- [x] `cd backend && go test ./internal/usecase/pipeline/...`
- [x] `cd backend && go test ./...`
- [x] `cd backend && make vet`

## Deploy verification (before commit — runtime only)
- [x] Build backend image locally with git SHA tag and push both SHA + `cloudrun-dev-latest`.
- [x] Deploy backend dev using the SHA-tagged image and record revision/image/URL.
- [x] Create a 100-asset batch from dev frontend/browser or equivalent frontend-origin API request.
- [x] Verify `GET /api/v1/backfill/{id}` converges to `status=completed`, `completedCount=100`, `failedCount=0`.
- [x] Verify `GET /api/v1/backfill/{id}/node-summary` reports logical totals and complete coverage for 100 items.
- [x] Verify `GET /api/v1/pipeline-runs?view=summary&batchJobId={id}&page=1&pageSize=20` reports total 100 in the default view.

### Dev verification evidence
- Backend dev revision: `cyber-databrew-backend-dev-00834-dz9`
- Image: `us-central1-docker.pkg.dev/green-valley-442103/cyber-databrew-images/cyber-databrew-backend:ce5fdf3-cyb2007-r2`
- Digest: `sha256:9237da2474e84c004e2206f622863e2327a04ffe64847937b54d6d9ad1445815`
- Dev URL: `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app`
- New 100-task batch from dev frontend origin: `b702e330-954b-434f-95fd-dfc6d4f9c4dd`
  - Job: `status=completed`, `totalCount=100`, `completedCount=100`, `failedCount=0`
  - Node summary: `subtasks.total=100`, `completed=100`, `failed=0`, `running=0`, `pending=0`, `runsWithNodeRows=100`, `runsTotal=100`, `complete=true`
  - Child list: `total=100`, `returned=100`, `Succeeded=100`, no `Error` rows
- Historical duplicate batch rechecked: `b9b5e409-73f5-47bf-b9c7-dd6556f0a7e3`
  - Job: `status=completed`, `completedCount=100`, `failedCount=0`
  - Node summary: `runsWithNodeRows=100`, `runsTotal=100`, `complete=true`
  - Child list: `total=100`, `Succeeded=100`, no `Error` rows
- Browser verification:
  - Batch detail `/pipeline/batch/b702e330-954b-434f-95fd-dfc6d4f9c4dd` renders the 100 child rows and `查看` actions.
  - Workflow detail `/pipeline/executions/e2e-preview-echo-20260613-c40f00` renders node details, logs, Pod, and event timeline.
  - Historical failure details verified with `node-failures?pipelineNodeId=step-step-1`: returns `assetId`, `workflowName`, node status, and message (`pod deleted`).

## PR
- [ ] PR template filled; Linear `CYB-2007` linked.
- [ ] Include root cause, test evidence, deploy revision, and 100-task regression evidence.
