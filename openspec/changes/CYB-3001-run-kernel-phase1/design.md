# Design — CYB-3001

## Architecture Context
- **Constraints**: Existing durable execution data lives in `pipeline_runs`, `pipeline_run_nodes`, `pipeline_run_events`, and backfill tables. New schema work is intentionally deferred.
- **Goals**: Make Run the product API and frontend entry point while keeping current runtime behavior compatible.
- **Non-Goals**: Removing workflow/deployment APIs, introducing new tables, or changing Argo submission semantics.

## Affected Modules
- `backend/routes/routes.go` — register `/runs` alongside legacy `/pipeline-runs`.
- `backend/internal/handlers/pipeline/handler.go` — add run-centric handlers and lightweight resource projections.
- `backend/internal/usecase/pipeline/usecase.go` — expose nodes, inputs, outputs, children, and runtime projections from current run data.
- `api/openapi.yaml` — document new run-centric API surface.
- `docs/review/api-guide.md` — explain Run/Workflow/Deployment semantics.
- `scripts/smoke-runs-dev.sh` — smoke the new dev run endpoints.
- `Frontend/src/api/runApi.ts` — new API client for run-centric code.
- `Frontend/src/App.tsx` / `WorkflowDetailPage.tsx` — add `/runs/:runId` route support.

## Architecture Decisions

### Decision 1: Map Run API to existing PipelineRun ledger in Phase 1
- **Approach**: Reuse `models.PipelineRun` and existing repositories while exposing `/runs` paths and run-centric helper views.
- **Alternative**: Create new `runs` tables immediately.
- **Rationale**: The current database already contains run identity, events, nodes, cost, target snapshot, and workflow refs. Alias-first reduces migration risk and lets frontend stop depending on workflowName sooner.
- **Trade-off**: JSON still contains compatibility fields such as `pipelineName` and `workflowName`.
- **Rollback**: Remove `/runs` route registrations and frontend `runApi` imports; legacy `/pipeline-runs` continues to work.

### Decision 2: Runtime details remain debug enrichment
- **Approach**: `/runs/:id/runtime` returns the runtime reference and current compatibility workflow fields from the Run record.
- **Alternative**: Proxy full Argo workflow detail through `/runs/:id/runtime`.
- **Rationale**: Product run detail should not fail when Argo TTL deletes the Workflow. Raw workflow detail remains available under `/workflows`.

### Decision 3: Inputs and outputs use projections before dedicated tables
- **Approach**: `/runs/:id/inputs` derives assets, config references, parameters, and runtime target from `PipelineRun` fields and pipeline JSON. `/runs/:id/outputs` derives logs/metrics/artifact placeholders from nodes and asset-node data.
- **Alternative**: Block Phase 1 until `run_inputs` and `run_outputs` tables exist.
- **Rationale**: The product API can stabilize now while preserving future storage changes behind the same contract.

## Data Flow

```text
Product UI
  -> Frontend runApi
  -> /api/v1/runs/*
  -> pipeline Handler run-centric methods
  -> pipeline Usecase
  -> pipeline_runs / pipeline_run_nodes / pipeline_run_events
  -> optional Argo runtime enrichment only when explicitly requested
```

## Data Model Changes
- None in Phase 1.

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Run API is initially an alias over PipelineRun | Some field names remain legacy-shaped | Document compatibility and keep new frontend imports in `runApi.ts` |
| `/runs/:id/children` cannot fully model Batch Run Tree yet | Batch parent aggregation is not complete | Return current batch-scoped child runs when `batchJobId` exists; defer relation table |
| `/runs/:id/inputs` config projection depends on pipeline JSON shape | Some legacy configs may be partial | Include source metadata and reasonable empty states |
