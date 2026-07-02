# Design — CYB-1565

## Architecture Context
- CYB-1564 adds `pipeline_run_events` and a polling watcher that records workflow/node/pod lifecycle events.
- Current `pipeline_run_nodes` stores node snapshots and estimated node cost when resource duration is available.
- Pipeline runs already carry `asset_ids`; no-asset runs are explicitly supported.
- The UI already has a placeholder for "资产 × 节点明细" and a compact run events card.

## Goals
- Build the next observability layer from DataBrew-owned records, not direct browser access to Argo/GKE.
- Keep P0.2 deployable in one PR by using estimated cost and configurable notification hooks.
- Avoid a large controller rewrite; improve polling with persisted watcher state and make watch-stream migration possible later.

## Non-Goals
- Exact cloud billing reconciliation.
- Full production notification routing and subscriptions.
- Replacing Argo polling with watch streams in this PR.

## Affected Modules
- `backend/internal/models/pipeline.go` — asset-node, cost summary, notification candidate, watcher state models.
- `backend/internal/repository/pipeline_repository.go` — new repository interfaces.
- `backend/internal/postgres/pipeline_repo.go` — asset-node and watcher state persistence.
- `backend/internal/usecase/pipeline/usecase.go` — derivation, cost summary, notification candidate emission, incremental sync.
- `backend/internal/handlers/pipeline/handler.go` — new list/detail APIs.
- `api/openapi.yaml`, `docs/review/api-guide.md`, `sdk/`, `scripts/` — API contract sync.
- `Frontend/src/api/pipelineApi.ts` — typed clients.
- `Frontend/src/pages/WorkflowDetailPage.tsx` / related components — matrix and full timeline UI.

## Architecture Decisions

### Decision 1: Asset-node snapshots are derived from run nodes first
- **Approach**: Generate `pipeline_run_asset_nodes` rows from `pipeline_runs.asset_ids` × `pipeline_run_nodes`; no-asset runs use one synthetic asset key such as `no-asset`.
- **Alternative**: Wait for each component to report per-asset state explicitly.
- **Rationale**: Current workflows mostly execute a node once per run; this gives immediate value while leaving explicit per-asset callbacks for later.
- **Trade-off**: Early P0.2 rows are "run-node scoped per asset" estimates, not true per-asset execution inside a pod.

### Decision 2: Cost remains estimated and source-labelled
- **Approach**: Expose cost fields with `costSource=estimated_resource_duration` or `not_available`.
- **Alternative**: Block until GCP Billing export integration exists.
- **Rationale**: Users need to identify expensive nodes now; exact billing can be reconciled later.
- **Trade-off**: UI must label costs as estimated.

### Decision 3: Notifications use candidates before provider integrations
- **Approach**: Persist or emit `pipeline_run_notification_candidates` for failed/error events with idempotency keys; provider delivery is disabled unless configured.
- **Alternative**: Implement Feishu webhook delivery immediately.
- **Rationale**: Notification correctness begins with exactly-once candidate creation; provider routing can be added safely later.
- **Trade-off**: P0.2 may show "notification candidate created" instead of real delivered message when no sink is configured.

### Decision 4: Polling watcher gets persisted sync state, not full Argo watch stream yet
- **Approach**: Store `pipeline_run_watcher_state` with last tick time, active scan limit, and last observed run/workflow cursor.
- **Alternative**: Build a long-running Argo watch stream controller now.
- **Rationale**: Cloud Run lifecycle and current code paths are already polling-friendly; persisted state reduces repeated work without creating a new controller subsystem.
- **Trade-off**: Realtime latency remains polling-bound.

## Data Flow

```text
CYB-1564 watcher / run detail refresh
  -> refresh pipeline_run_nodes
  -> append pipeline_run_events
  -> derive pipeline_run_asset_nodes from run assets × nodes
  -> compute run/node/asset-node estimated cost summary
  -> create notification candidates for failed/error events

frontend workflow detail
  -> GET run by workflow name/list
  -> GET events with filters/search/cursor
  -> GET asset-nodes
  -> GET cost-summary
  -> render matrix + timeline + drill-down actions
```

## Data Model Changes

### `pipeline_run_asset_nodes`
- `id`
- `run_id`
- `asset_id`
- `pipeline_node_id`
- `argo_node_id`
- `display_name`
- `status`
- `message`
- `pod_name`
- `log_ref`
- `estimated_cost_usd`
- `cost_source`
- `started_at`
- `finished_at`
- `updated_at`

### `pipeline_run_notification_candidates`
- `id`
- `run_id`
- `event_id`
- `event_type`
- `subject_type`
- `subject_id`
- `status`
- `message`
- `sink_type`
- `delivery_status`
- `idempotency_key`
- `created_at`

### `pipeline_run_watcher_state`
- `id`
- `last_synced_at`
- `active_scan_limit`
- `last_error`
- `updated_at`

## API Shape

`GET /api/v1/pipeline-runs/{id}/asset-nodes`
- Query: `assetId`, `nodeId`, `status`, `limit`, `cursor`.
- Response: `{items, nextCursor, total, summary}`.

`GET /api/v1/pipeline-runs/{id}/cost-summary`
- Response: run total, node summaries, asset-node summaries, source labels.

`GET /api/v1/pipeline-runs/{id}/events`
- Add query: `q`, `status`, `from`, `to`.
- Existing response remains backward compatible.

## Frontend Behavior
- Compact top cards continue to show latest context.
- Full timeline area supports filter chips, text search, and load-more.
- Asset-node panel shows a dense table/matrix with status, asset, node, duration, estimated cost, log/pod/monitor/debug actions.
- Costs are marked as estimated when not backed by billing export.
- Notification candidates are shown as audit facts only if backend exposes them; no fake "sent" state.

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Asset-node rows imply more precision than available | Users may over-trust estimates | Label source and show no-asset/run-node scope clearly |
| More schema in one PR | Migration/deploy risk | Keep tables append/upsert-oriented and smoke after migration |
| Watcher state still polling-bound | Not instant at high scale | Make interval/limit tunable and document Argo watch stream as next evolution |
| Notification provider not configured | Users expect external alerts | Surface candidate/audit only; delivery config is explicit |
