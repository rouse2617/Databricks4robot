# Design — CYB-3002

## Architecture Context
- **Constraints**: Phase two must build on the current `pipeline_runs`, `backfill_jobs`, and `backfill_items` tables. New relation tables are deferred.
- **Goals**: Make Batch/Backfill visible as Run Tree structure while preserving current execution behavior.
- **Non-Goals**: Removing legacy APIs, changing Argo submission semantics, or adding a database migration.

## Affected Modules
- `backend/internal/runtimeos/state/` — add pure RunStateMachine and Run Tree aggregation helpers.
- `backend/internal/models/pipeline.go` — extend Run child response models with relation and summary projections.
- `backend/internal/usecase/pipeline/usecase.go` — return richer children response and reuse RunStateMachine aggregation.
- `backend/internal/usecase/pipeline/batch_subtask.go` — preserve child Run relation metadata through existing `BatchJobID`.
- `backend/internal/usecase/backfill/usecase.go` — materialize parent Run records for batch/backfill jobs when possible.
- `backend/internal/handlers/pipeline/handler.go` — keep handlers thin and return the enriched model.
- `api/openapi.yaml` / `docs/review/api-guide.md` / `scripts/smoke-runs-dev.sh` — sync API contract and dev smoke.
- `Frontend/src/api/runApi.ts` — type the enriched children response.
- `Frontend/src/pages/BatchJobDetailPage.tsx` — show Run Tree summary and child Run links using `runApi`.

## Architecture Decisions

### Decision 1: Project RunRelation before creating a table
- **Approach**: Derive `batch_child` relation rows from `PipelineRun.BatchJobID == parentRunID`.
- **Alternative**: Add a `run_relations` table now.
- **Rationale**: Existing child Runs already carry the parent batch id. A projection proves API/UI semantics without taking migration risk.
- **Trade-off**: Relation history is limited to relationships represented by current fields.
- **Rollback**: Remove relation fields from the response and keep returning the current child Run list.

### Decision 2: Add RunStateMachine as a pure helper first
- **Approach**: Introduce a pure status normalizer and child aggregation helper under `runtimeos/state`.
- **Alternative**: Inline aggregation in `pipeline.Usecase.ListRunChildren`.
- **Rationale**: The Runtime OS target needs a single place for Run status semantics. Starting pure keeps it easy to test and avoids broad side effects.
- **Trade-off**: The helper will initially be used by children aggregation only; later work can move more status transitions into it.
- **Rollback**: Replace the helper call with previous direct child list behavior.

### Decision 3: Parent Run materialization reuses existing PipelineRun storage
- **Approach**: When a backfill/batch job is created, create or preserve a parent PipelineRun with id equal to the job id and no runtime workflow requirement.
- **Alternative**: Treat BackfillJob itself as the parent response and bypass `/runs/:id`.
- **Rationale**: `/runs/:id/children` must work from product Run identity, and phase one already maps Run to PipelineRun storage.
- **Trade-off**: Parent Runs will still have compatibility-shaped fields until a future schema migration adds `run_type`.
- **Rollback**: Stop creating parent Runs; legacy batch/backfill pages continue to use existing job tables.

## Data Flow

```text
Backfill / Batch create
  -> BackfillJob is saved
  -> parent PipelineRun is ensured with id = job id
  -> child PipelineRuns continue to use BatchJobID = job id
  -> GET /runs/:id/children
  -> RunStateMachine aggregates children
  -> API returns { items, relations, summary }
  -> Batch detail renders Run Tree summary and child Run links
```

## Data Model Changes
- No database schema changes.
- API model additions:
  - `RunRelation` projection.
  - `RunChildSummary` projection.
  - `RunChildList.relations` and `RunChildList.summary`.

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Parent Run records are compatibility-shaped | UI may still see fields like `pipelineName` and `workflowName` | Mark parent runtime as optional and avoid requiring workflowName for parent Runs |
| Existing historical batch jobs may not have parent Runs | `/runs/:id` could 404 for older jobs until reconciled | Batch detail can fall back to job data; children projection only promises parent Run behavior for newly materialized jobs in this phase |
| Aggregation policy can be debated | Operators may expect failed+running to display failed | Expose detailed counts and document deterministic aggregate priority |
