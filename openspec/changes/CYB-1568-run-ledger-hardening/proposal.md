# Proposal — CYB-1568

## Why
Pipeline execution history must remain useful after Argo objects are cleaned up, and users need a trustworthy DataBrew run timeline for debugging failures, retries, deletes, and asset-node progress.

## What Changes

### New Capabilities
- DataBrew records a complete durable run timeline for submit, schedule, workflow creation, Pod creation, node start/end/failure, retry, resubmit, and delete events.
- Watcher health is visible and persistent, including last scan, last success, last error, retry counters, scan lag, and active-run scan limits.
- Run detail pages can rely on DataBrew ledger data when Argo workflows are expired or unavailable.
- Run status and node status are reconciled into `pipeline_runs`, `pipeline_run_nodes`, and `pipeline_run_events` as the product source of truth.

### Modified Capabilities
- Existing PR #97 run event ingestion becomes a hardened ledger instead of a best-effort timeline.
- Existing polling watcher remains the default implementation, but gains bounded retry and repair behavior.
- Workflow delete/retry/resubmit operations record explicit timeline/audit events before side effects.

## Impact
- **Affected code**: `backend/internal/models`, `backend/internal/repository`, `backend/internal/postgres`, `backend/internal/usecase/pipeline`, `backend/internal/handlers/pipeline`, `Frontend/src/api`, `Frontend/src/pages`
- **New APIs**:
  - `GET /api/v1/pipeline-runs/watcher/status`
- **Changed APIs**:
  - `GET /api/v1/pipeline-runs/{id}/events` may include additional event types and reason/payload details.
  - `GET /api/v1/pipeline-runs/{id}` should prefer stored run/node data when Argo is expired.
- **Schema**: migration required to extend watcher state and optionally add event classification indexes.

## Scope
- **In scope**:
  - Define and persist stable event types for run, workflow, node, pod, retry, resubmit, and delete lifecycle.
  - Add watcher health fields and repository support.
  - Add a watcher health/status endpoint for frontend and smoke tests.
  - Add missed-scan repair behavior for active and recently terminal runs.
  - Preserve historical run/node/event details after Argo cleanup.
  - Update execution detail UI empty states to distinguish "Argo expired, showing DataBrew history" from missing data.
  - Update OpenAPI, API guide, smoke, and tests.
- **Out of scope**:
  - Replacing polling with Argo watch stream.
  - Implementing terminal/exec debug sessions.
  - Actual GCP Billing reconciliation.
  - External notification delivery beyond existing notification candidates.

## Success Criteria
- [ ] Completed/failed/deleted runs retain a meaningful DataBrew timeline even if Argo no longer returns the workflow.
- [ ] Watcher status shows last scan, success/error, retry counts, lag, and active scan limit.
- [ ] Retry/resubmit/delete operations create durable run events with idempotency.
- [ ] Run detail event timeline supports the new event types with filter/search/pagination.
- [ ] A dev smoke can create or use a run, sync watcher, and verify timeline events from DataBrew storage.

## Goals (SLO)
- **Latency**: active run state changes visible in the ledger within 10 seconds under default polling.
- **Concurrency**: watcher handles at least 100 active runs with bounded work per tick.
- **Quality**: targeted backend tests for event idempotency/watcher health plus frontend and dev smoke verification.
