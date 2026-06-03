# Proposal — CYB-1613

## Why
Workflow 详情页只返回 Argo `status.nodes` 中已经创建的节点，导致运行中的多节点 DAG 逐步“长出来”，用户无法在打开页面时看到完整流水线拓扑和未执行步骤。

## What Changes

### New Capabilities
- Workflow detail returns a complete DAG view by merging runtime nodes from `status.nodes` with static DAG tasks from `spec.templates`.
- Static-only DAG tasks are returned as pending/not-yet-started nodes so the frontend can render the full graph immediately.

### Modified Capabilities
- Workflow DAG edges include static dependencies from `spec.templates[].dag.tasks` even when downstream tasks do not yet exist in `status.nodes`.

## Impact
- **Affected code**: `backend/internal/handlers/workflow/handler.go`, `backend/internal/handlers/workflow/dag_edges.go`, `backend/internal/handlers/workflow/handler_test.go`
- **New APIs**: none. Existing `GET /api/v1/workflows/{name}` response includes additional pending nodes/edges for existing fields.
- **Dependencies**: none.

## Scope
- **In scope**: merge static DAG tasks with runtime nodes, stable pending node IDs, static dependency edges, unit tests.
- **Out of scope**: frontend graph redesign, Argo workflow execution behavior, persisted pipeline template recovery, nested DAG visualization beyond current handler conventions.

## Success Criteria
- [ ] A workflow with four DAG tasks but only one or two runtime status nodes returns all four tasks in `nodes`.
- [ ] Not-yet-created tasks have a clear pending phase and stable ID matching static DAG edges.
- [ ] Runtime nodes keep pod name, logs, debug, timing, phase, and output metadata.
- [ ] Existing runtime/fallback edge behavior remains compatible.
