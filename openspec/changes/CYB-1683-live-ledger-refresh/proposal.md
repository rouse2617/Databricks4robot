# Proposal — CYB-1683

## Why

Workflow detail can show live Argo DAG nodes as completed while the DataBrew asset-node table still shows stale pending rows. This makes successful runs look like they are still waiting for resource snapshots.

## What Changes

- Refresh run events, asset-node rows, and cost summary during active workflow polling.
- Do extra quiet ledger refreshes after a workflow transitions from active to terminal.
- Use live Argo node status as a display fallback when DataBrew asset-node rows lag behind.

## Impact

- **Affected code**: `Frontend/src/pages/useWorkflowDetail.ts`, `Frontend/src/pages/WorkflowDetailPage.tsx`, `Frontend/src/pages/WorkflowExecutionList.tsx`.
- **New APIs**: none.
- **Dependencies**: none.

## Scope

- **In scope**: execution detail live refresh, stale asset-node display fallback, focused frontend tests.
- **Out of scope**: backend ledger write-back changes, Argo controller behavior, cost calculation changes.
