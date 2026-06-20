# Proposal — CYB-3002

## Why
Runtime OS phase one made `/runs` the product execution entry point, but Batch and Backfill still behave like side lists instead of parent Runs with child Runs. Operators need a stable Run Tree view and a deterministic parent status summary before deeper scheduler and relation-table work.

## What Changes

### New Capabilities
- Run Tree projection for batch/backfill execution records using existing child Run data.
- Run child status aggregation so a parent Run view can explain whether children are pending, running, succeeded, or failed.
- RunRelation projection for `batch_child` relationships without requiring a schema migration.

### Modified Capabilities
- `/runs/:id/children` becomes a structured Run Tree response with child Runs, relation metadata, and aggregate summary.
- Batch detail UI can consume Run Tree data instead of treating batch child execution rows as a disconnected list.
- Run Kernel gains a small state-machine boundary for status normalization and child aggregation.

## Impact
- **Affected code**:
  - `backend/internal/runtimeos/state/`
  - `backend/internal/usecase/pipeline/usecase.go`
  - `backend/internal/usecase/pipeline/batch_subtask.go`
  - `backend/internal/usecase/backfill/usecase.go`
  - `backend/internal/models/pipeline.go`
  - `backend/internal/handlers/pipeline/handler.go`
  - `api/openapi.yaml`
  - `docs/review/api-guide.md`
  - `scripts/smoke-runs-dev.sh`
  - `Frontend/src/api/runApi.ts`
  - `Frontend/src/pages/BatchJobDetailPage.tsx`
- **New APIs**: none.
- **Changed APIs**: `GET /api/v1/runs/:id/children` response shape gains optional `relations` and `summary`.
- **Dependencies**: none.

## Scope
- **In scope**:
  - Derive parent/child Run relationships from existing batch job ids.
  - Ensure newly created batch/backfill jobs have a parent Run record when possible.
  - Aggregate child statuses through a RunStateMachine helper.
  - Keep legacy batch/backfill APIs working.
  - Update frontend/API docs/tests for the richer children response.
- **Out of scope**:
  - New `run_relations` database table.
  - Removing BackfillJob, BackfillItem, Deployment, Workflow, or pipeline-runs APIs.
  - Full Backfill scheduler rewrite.
  - New long-term Run schema migration.

## Success Criteria
- [ ] `/runs/:id/children` returns child Runs and `batch_child` relation projections for a batch parent Run.
- [ ] `/runs/:id/children` returns an aggregate child status summary with deterministic parent status.
- [ ] New batch/backfill creation paths materialize or preserve a parent Run identity that can be inspected through `/runs/:id`.
- [ ] Batch detail UI can display child Run Tree summary without calling legacy pipeline-run paths for product child rows.
- [ ] Existing legacy backfill, deployment, workflow, and pipeline-run endpoints remain compatible.
- [ ] Backend and frontend targeted tests cover Run Tree status aggregation and response typing.

## Goals (SLO)
- **Latency**: `GET /runs/:id/children` remains bounded to the existing 500 child Run page and avoids live Argo fan-out.
- **Concurrency**: No new background scheduler goroutine class is introduced in this phase.
- **Quality**: Tests cover empty, pending/running, all-success, and failure aggregation cases.
