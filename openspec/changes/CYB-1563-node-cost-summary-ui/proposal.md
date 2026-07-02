# CYB-1563 — Node Cost Summary UI

## Problem

Pipeline execution details currently show the DAG and per-node detail drawer, but
users cannot quickly answer which node was slowest, most expensive, or most worth
optimizing.

CyberPipe's History page exposes a compact "cost by node" summary. DataBrew
needs the same decision surface in its own pipeline execution detail experience.

## Goal

Add a frontend P0 version of a "node duration / cost" summary to
`WorkflowDetailPage`, using the data that already exists in the workflow detail
response:

- node name
- node phase
- pod presence
- started / finished time
- resource duration
- optional estimated cost from workflow node data or first-class pipeline run
  records

## Scope

This change is frontend-led and uses existing backend run data:

- Add a node summary panel above the DAG / timeline view.
- Derive duration from `startedAt` and `finishedAt` where available.
- Display estimated cost when present on the node payload; otherwise show a clear
  "pending backend data" value.
- Support sorting by total duration and estimated cost.
- Let users click a row to open the existing node detail panel.
- Fetch existing `GET /api/v1/pipeline-runs` data on the executions list and
  merge run-level `totalEstimatedCost` by workflow name.
- Add backend handler tests proving pipeline run responses include node-derived
  `totalEstimatedCost`.

## Out Of Scope

- No new backend route changes.
- No new database migrations.
- No GCP Billing export integration.
- No asset x node persistence.
- No run_events watcher.

## Linear

CYB-1563
