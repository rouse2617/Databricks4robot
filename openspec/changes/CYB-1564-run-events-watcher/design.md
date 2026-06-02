# Design — CYB-1564

## Architecture Context
- DataBrew currently has `pipeline_runs` and `pipeline_run_nodes` as first-class execution records.
- Argo workflow status is useful for current state, but workflow objects and Pods can be TTL-cleaned, leaving users without a durable explanation of what happened.
- The current workflow detail page has a "运行事件" placeholder, so the product already reserves space for this capability.
- The deployment environment is GCP Cloud Run backend talking to Argo/GKE in configured namespaces.
- Browser clients must not call Argo or Kubernetes directly.

## Goals
- Keep a DataBrew-owned chronological event ledger for each pipeline run.
- Make event ingestion restart-safe and idempotent.
- Keep P0.1 small enough to ship before asset × node persistence and billing reconciliation.
- Preserve existing run list/detail behavior while adding durable events.

## Non-Goals
- Streaming every Kubernetes event in real time.
- Building notification delivery.
- Replacing `pipeline_run_nodes` cost/resource summaries.
- Implementing final GCP billing-grade cost attribution.

## Affected Modules
- `backend/internal/models/pipeline.go` — add `PipelineRunEvent` domain model and event enums.
- `backend/internal/repository/pipeline_repository.go` — add event repository interface.
- `backend/internal/postgres/pipeline_repo.go` — persist/list events and idempotent inserts.
- `backend/internal/usecase/pipeline/usecase.go` — append events during run lifecycle and watcher sync.
- `backend/internal/handlers/pipeline/handler.go` — expose event list API.
- `backend/routes/routes.go` — register `GET /api/v1/pipeline-runs/:id/events`.
- `backend/cmd/server/core.go` — wire repository/usecase dependencies and watcher startup if enabled.
- `api/openapi.yaml`, `docs/review/api-guide.md`, `sdk/`, `scripts/` — API contract sync.
- `Frontend/src/api/pipelineApi.ts` — typed event client.
- `Frontend/src/pages/WorkflowDetailPage.tsx` and `Frontend/src/pages/useWorkflowDetail.ts` — timeline consumption and states.

## Architecture Decisions

### Decision 1: Use a polling watcher before Argo watch streams
- **Approach**: A backend watcher periodically loads active DataBrew pipeline runs, gets their Argo workflows, diffs observed workflow/node/pod state, and appends idempotent events.
- **Alternative**: Maintain long-lived Argo watch streams from Cloud Run.
- **Rationale**: Polling matches the current Argo integration, is easier to restart, and works in Cloud Run without managing stream lifecycle edge cases.
- **Trade-off**: Event freshness is bounded by the polling interval, not instant.
- **Rollback**: Disable watcher startup and keep direct run create/retry/delete events.

### Decision 2: Store run events separately from asset events
- **Approach**: Create `pipeline_run_events` for execution event history.
- **Alternative**: Reuse `asset_events`.
- **Rationale**: Asset events are an asset audit/outbox concept; run events need workflow/node/pod subjects, idempotency keys, and execution timeline ordering.
- **Trade-off**: Adds one table and repository surface.
- **Rollback**: Drop route and watcher consumers while keeping table unused until migration rollback is approved.

### Decision 3: Append immutable facts with idempotency keys
- **Approach**: Each event row has immutable facts plus an `idempotency_key`; repository insert uses conflict ignore/return existing behavior.
- **Alternative**: Mutate a single event row as phases change.
- **Rationale**: Timeline and audit use cases need historical facts, while idempotency prevents duplicated watcher observations.
- **Trade-off**: More rows than a mutable status table.

### Decision 4: Timeline API returns chronological pages
- **Approach**: `GET /api/v1/pipeline-runs/{id}/events` returns events sorted by `occurredAt`, `sequence`, with cursor support for additional pages or incremental loading.
- **Alternative**: Return newest-first like many audit lists.
- **Rationale**: Run detail is a timeline; chronological ordering avoids frontend reordering and makes retries/status transitions readable.
- **Trade-off**: Loading latest-only views may need a future query option.

## Event Model

Event fields:
- `id`: stable event ID.
- `runId`: DataBrew pipeline run ID.
- `workflowName`: Argo workflow name when known.
- `eventType`: machine-readable event type.
- `subjectType`: `run`, `workflow`, `node`, or `pod`.
- `subjectId`: workflow name, Argo node ID, Pod name, or run ID.
- `status`: optional phase/status associated with the event.
- `message`: short user-facing explanation.
- `reason`: Argo/Kubernetes reason when available.
- `occurredAt`: source event time when known, otherwise observation time.
- `observedAt`: backend observation time.
- `payload`: small JSON payload for source-specific fields.
- `idempotencyKey`: deterministic key used to prevent duplicates.
- `sequence`: monotonic row sequence for pagination and stable timeline ordering.

Initial event types:
- `run_submitted`
- `workflow_observed`
- `workflow_phase_changed`
- `node_started`
- `node_succeeded`
- `node_failed`
- `node_error`
- `pod_created`
- `pod_phase_changed`
- `run_completed`
- `run_failed`
- `run_retry_requested`
- `run_resubmitted`
- `run_stop_requested`
- `run_deleted`

## Data Flow

```text
user action / scheduler
  -> pipeline usecase creates or mutates pipeline_run
  -> append operator/run event
  -> Argo workflow submitted or changed

watcher tick
  -> list active pipeline_runs
  -> get Argo workflow by workflow_name
  -> compare workflow/node/pod phases against existing event keys
  -> append new pipeline_run_events
  -> refresh pipeline_run / pipeline_run_nodes current summaries

frontend detail page
  -> get workflow/run detail
  -> get /pipeline-runs/{id}/events
  -> render chronological timeline with refresh/error/empty states
```

## Data Model Changes
- **Table**: `pipeline_run_events`
- **Change**: add durable event ledger keyed by `run_id` and `idempotency_key`.
- **Migration**: required under `backend/migrations/`, which is an off-limits path and needs explicit user approval before implementation.
- **Indexes**:
  - `(run_id, sequence)` for timeline pages.
  - `(run_id, idempotency_key)` unique for deduplication.
  - `(workflow_name, occurred_at)` for investigation and watcher support.

## API Shape

`GET /api/v1/pipeline-runs/{id}/events`

Query:
- `limit`: optional, default 100, max 500.
- `cursor`: optional sequence cursor.
- `subjectType`: optional filter.
- `eventType`: optional filter.

Response:
- `items`: ordered list of `PipelineRunEvent`.
- `nextCursor`: optional cursor for more events.
- `total`: optional total count when cheap to compute.

Error semantics:
- `400 INVALID_ARGUMENT`: invalid limit/cursor/filter.
- `404 NOT_FOUND`: run does not exist.
- `500 INTERNAL_ERROR`: unexpected backend failure.

## Frontend Behavior
- The run detail summary card uses real events in the "运行事件" panel.
- Timeline rows show time, type, subject, status, and message.
- Empty state says no events have been recorded yet, not "接口待接入".
- Error state keeps the rest of the detail page usable and exposes retry/refresh.
- Clicking node-related events selects the node detail panel when the node exists in the DAG.

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Watcher duplicates events after restart | Timeline becomes noisy | Deterministic idempotency keys and unique constraint |
| Argo workflow already deleted before first watcher sync | Missing workflow/node events | Direct run lifecycle events still exist; future retention can query archives |
| Poll interval too slow for live debugging | Users wait several seconds | Keep manual refresh; tune dev interval; future stream watcher can replace polling |
| Migration timing mismatch during deploy | Runtime 500 for new API/table | Apply approved migration before backend deploy and smoke the new endpoint |
| Event payload grows too large | Storage and API bloat | Keep payload small and store only diagnostic fields needed for timeline |
