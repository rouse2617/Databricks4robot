# Proposal — CYB-1565

## Why
Users need pipeline execution observability beyond a run-level event list: they need to know which asset failed at which node, which node is slow or expensive, and how to inspect logs/resources without relying on transient Argo objects.

## What Changes

### New Capabilities
- Pipeline run details expose an asset × node matrix with per-cell status, log/pod/resource links, timestamps, and estimated cost.
- Pipeline run details expose a fuller run event timeline with filters, search, pagination/load-more, and node/asset drill-down.
- Backend exposes estimated cost/audit summaries at run, node, and asset-node levels using existing resource duration data.
- Failure events can be surfaced through a notification extension point so future Feishu/webhook integrations do not require changing watcher logic.
- Watcher synchronization gains an incremental cursor/state model while keeping polling as the fallback path.

### Modified Capabilities
- Run event ingestion from CYB-1564 becomes the source for asset-node derivation, audit summaries, and notifications.
- Workflow detail UI separates "latest event summary" from the full timeline view.

## Impact
- **Affected code**: `backend/internal/models`, `backend/internal/repository`, `backend/internal/postgres`, `backend/internal/usecase/pipeline`, `backend/internal/handlers/pipeline`, `Frontend/src/api`, `Frontend/src/pages`, `sdk/`
- **New APIs**:
  - `GET /api/v1/pipeline-runs/{id}/asset-nodes`
  - `GET /api/v1/pipeline-runs/{id}/cost-summary`
  - Query additions to `GET /api/v1/pipeline-runs/{id}/events`
- **Dependencies**: no new third-party dependency planned
- **Schema**: new persistence for asset-node snapshots and watcher cursor/state; migration required

## Scope
- **In scope**:
  - Store and list asset-node execution snapshots.
  - Derive no-asset and asset-backed rows from `pipeline_runs`, `pipeline_run_nodes`, and event payloads.
  - Provide estimated cost/audit summary APIs using existing estimated node costs.
  - Add timeline filtering/search/load-more UI.
  - Add notification extension point for failed/error events, with disabled-by-default config.
  - Add watcher cursor/state persistence so polling only scans active runs and records last sync time.
  - Update OpenAPI, API guide, SDK, smoke, and tests.
- **Out of scope**:
  - Exact GCP Billing reconciliation.
  - Production Feishu credential/config rollout.
  - Full Argo watch stream controller replacing polling.
  - Long-term retention/archive policy.

## Success Criteria
- [ ] A run detail page shows asset × node rows for asset-backed runs and no-asset rows for debug runs.
- [ ] Each asset-node row links to existing node logs, Pod diagnostics, monitoring, and cost detail affordances when available.
- [ ] Users can sort or visually identify slow/expensive nodes and asset-node rows.
- [ ] The full run event timeline supports status/type/subject filters, text search, and load-more.
- [ ] Failed/error events create a notification candidate record or invoke the configured notification sink exactly once.
- [ ] Watcher sync can resume after restart without rescanning all historical completed runs.

## Goals (SLO)
- **Latency**: active run asset-node/event updates visible within 10 seconds under default polling.
- **Concurrency**: handle at least 100 active runs with bounded watcher work per tick.
- **Quality**: targeted backend, frontend, SDK tests plus dev smoke coverage before PR merge.
