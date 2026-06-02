# Design

## Current State

Latest `origin/dev` already includes:

- `pipeline_runs` / `pipeline_run_nodes` tables.
- `pipeline_run_nodes.estimated_cost_usd`.
- resource usage endpoints for workflow and node resources.
- `WorkflowNodeDetailPanel` with summary, logs, runtime, IO, monitoring, and
  billing shells.

The missing user-facing piece is a run-level table that summarizes all nodes in
one place.

## Frontend Shape

Add a `WorkflowNodeSummaryTable` component near the top of
`WorkflowDetailPage`, after the run context panel and before DAG/timeline.

Rows are derived from `workflow.nodes`.

Columns:

- Node
- Status
- Pod
- Node duration
- Estimated cost
- Started
- Finished
- Actions

Actions:

- Logs opens the existing node logs modal.
- Runtime opens the existing node detail panel on the runtime tab.
- Billing opens the existing node detail panel on the runtime tab, where the
  billing section already exists.

## Data Derivation

Duration:

- If `startedAt` and `finishedAt` exist, calculate wall-clock seconds.
- If still running and `startedAt` exists, calculate from now.
- For terminal nodes without `finishedAt`, do not calculate against current
  time; fallback to `estimatedDuration` when present, otherwise show empty
  duration.

Cost:

- Use future-compatible optional node fields:
  - `estimatedCostUsd`
  - `cost.totalCostUsd`
- On the executions list, merge existing `pipeline_runs.totalEstimatedCost` by
  workflow name when available.
- If missing, show `待接入` or a quiet empty value.

Pod count:

- P0 frontend has no grouped summary API. Treat node with `podName` as one pod.
- For DAG/container-only nodes without podName, show zero or dash.

## Verification

- Frontend lint.
- Related tests for `WorkflowDetailPage` / new component.
- Frontend build.
- Local browser check with Chrome DevTools MCP.
