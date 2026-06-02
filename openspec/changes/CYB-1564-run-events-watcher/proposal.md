# Proposal — CYB-1564

## Why
DataBrew needs its own durable run event ledger so users can see a complete execution timeline after Argo workflow state or Pods are cleaned up.

## What Changes

### New Capabilities
- Pipeline execution details expose a chronological run event timeline for submission, workflow observation, node transitions, Pod creation/status changes, retries, and terminal outcomes.
- Backend persists pipeline run events independently from transient Argo workflow state.
- Backend watcher/poller observes active Argo workflows and appends idempotent run events.
- Frontend replaces the current placeholder run-events card with real timeline data and loading/error/empty states.

### Modified Capabilities
- Pipeline run creation, retry, stop, and delete flows record first-class events instead of only updating the run status.
- Pipeline run details no longer rely only on the current Argo workflow object to explain what happened.

## Impact
- **Affected code**: `backend/internal/models`, `backend/internal/repository`, `backend/internal/postgres`, `backend/internal/usecase/pipeline`, `backend/internal/handlers/pipeline`, `backend/routes`, `Frontend/src/api`, `Frontend/src/pages`
- **New APIs**: `GET /api/v1/pipeline-runs/{id}/events`
- **Dependencies**: no new third-party dependency planned
- **Schema**: new `pipeline_run_events` table, requiring an approved migration before implementation

## Scope
- **In scope**:
  - Persist run-level, workflow-level, node-level, and Pod-level event records.
  - Add a restart-safe Argo polling watcher for active runs.
  - Deduplicate events with deterministic idempotency keys.
  - Add paginated API access for a run's events.
  - Render the execution detail "运行事件" area as a real chronological timeline.
  - Update OpenAPI, API guide, SDK surface, smoke script, frontend API types/hooks, and tests in the same PR.
- **Out of scope**:
  - Full asset × node persistence and UI table.
  - Exact GCP Billing reconciliation.
  - Feishu/webhook notification delivery.
  - Replacing all existing workflow polling with watch streams.
  - Long-term event retention policy and archival.

## Success Criteria
- [ ] Starting a pipeline run records a `run_submitted` event that appears on the run detail page.
- [ ] A completed workflow records workflow and node terminal events, and the timeline remains available from DataBrew storage.
- [ ] A failed workflow records a failure event with the best available Argo message.
- [ ] Retrying or deleting a run records an operator action event.
- [ ] Duplicate watcher polling does not create duplicate events for the same observed transition.
- [ ] The new API is documented, typed, smoke-tested, and covered by backend/frontend tests.

## Goals (SLO)
- **Latency**: active run event synchronization should be visible within 10 seconds under normal dev polling.
- **Concurrency**: support at least 100 active workflows without requiring frontend list/get polling for every event.
- **Quality**: API contract, repository, usecase, and frontend timeline behavior covered by targeted tests.
