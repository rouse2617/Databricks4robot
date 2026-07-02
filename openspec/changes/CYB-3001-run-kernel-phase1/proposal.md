# Proposal — CYB-3001

## Why

DataBrew execution UX still exposes pipeline runs, Argo workflows, deployments, and batch items as competing product identities. Users need one stable product execution entity: Run.

## What Changes

### New Capabilities
- Product clients can use `/api/v1/runs` as the run-centric execution API.
- Users can open a run by `runId` via `/runs/:runId`.
- Existing workflow-name execution links resolve to their owning Run when ledger data exists.

### Modified Capabilities
- `/pipeline-runs` remains as a compatibility API backed by the same data.
- `/workflows` remains runtime debug and enrichment, not the preferred product list API.
- Frontend API code gets a run-centric client so new execution UI work does not import pipeline-run endpoints.

## Impact
- **Affected code**: `backend/routes`, `backend/internal/handlers/pipeline`, `backend/internal/usecase/pipeline`, `Frontend/src/api`, `Frontend/src/App.tsx`, `Frontend/src/pages/WorkflowDetailPage.tsx`
- **New APIs**: `GET/POST/DELETE /api/v1/runs`, run subresources under `/api/v1/runs/:id`, and operation aliases under `/api/v1/runs/:id/*`
- **Dependencies**: none

## Scope
- **In scope**: Phase 1 run API aliases, minimal run input/output/children/runtime/node views using current `PipelineRun` ledger data, OpenAPI/api-guide sync, smoke coverage, frontend `runApi`, and `/runs/:runId` route.
- **Out of scope**: new database tables, destructive migration, replacing all existing WorkflowExecutionList data flow, removing legacy `/deployments`, removing `/workflows`, and full Batch Run Tree persistence.

## Success Criteria
- [ ] `/api/v1/runs` list/get/by-workflow works and returns current run records.
- [ ] `/api/v1/runs/:id/events`, `/nodes`, `/asset-nodes`, `/cost-summary`, `/inputs`, `/outputs`, `/children`, and `/runtime` return deterministic JSON.
- [ ] `/api/v1/runs/:id/stop`, `/retry`, and `/delete` preserve existing ledger-event behavior.
- [ ] Unsupported or deferred run operations return explicit compatibility responses rather than pretending to be implemented.
- [ ] Frontend exposes `runApi.ts` and a `/runs/:runId` route without breaking legacy workflow links.
- [ ] Legacy `/pipeline-runs`, `/deployments`, and `/workflows` routes remain registered and tested.

## Goals (SLO)
- **Latency**: run list/get aliases add no extra database round-trips beyond the current pipeline-run handlers.
- **Quality**: backend handler tests and frontend API route tests cover the new run-centric paths.
