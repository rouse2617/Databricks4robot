# Proposal — CYB-1793

## Why
Workflow execution DAGs are technically correct but visually awkward for fanout/fanin workflows: join nodes look mechanically placed, incoming edges route around the graph, and read-only run detail nodes still show edit-style handles.

## What Changes

### New Capabilities
- None.

### Modified Capabilities
- Workflow execution DAG view SHALL make dependency and join semantics visually legible for fanout/fanin graphs.
- Read-only workflow execution DAG nodes SHALL avoid edit-mode affordances that imply the user can create or move edges.
- Multi-input join nodes SHALL be positioned and routed as convergence points rather than ordinary downstream nodes.

## Impact
- **Affected code**: `Frontend/src/pages/WorkflowDagView.tsx`, `Frontend/src/pages/WorkflowDagView.css`, `Frontend/src/pages/WorkflowDagNode.tsx`, related tests.
- **New APIs**: None.
- **Dependencies**: None planned.

## Scope
- **In scope**: Improve visual layout for the existing execution DAG view.
- **In scope**: Add deterministic layout tests for fan-in/join graphs.
- **In scope**: Preserve current node actions, search, selection, pan/zoom, and fitView behavior.
- **Out of scope**: Changing workflow execution semantics, pipeline designer port semantics, Argo/DAG backend shape, or adding real data-flow binding.
- **Out of scope**: Replacing the graph engine with a new package unless the existing approach cannot meet the acceptance criteria.

## Success Criteria
- [ ] On `qa-complex-dag-20260607041627-e3d3c7`, `qa-join-abc` reads as the convergence point for `qa-left-b`, `qa-right-b`, and `qa-mid-a`.
- [ ] Incoming join edges are easier to follow and do not imply hidden intermediate nodes.
- [ ] Read-only execution DAG nodes do not show misleading edit handles.
- [ ] DAG node click, action buttons, search, fitView, and node detail interactions continue to work.
- [ ] Chrome DevTools MCP verifies the deployed dev UI on the complex DAG URL.

## Goals (SLO)
- **Latency**: Layout computation remains client-side and deterministic for normal run detail graphs.
- **Concurrency**: No change.
- **Quality**: Targeted tests cover fan-in layout and read-only DAG affordances.
