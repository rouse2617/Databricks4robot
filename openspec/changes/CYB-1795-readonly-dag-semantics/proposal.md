# Proposal — CYB-1795

## Why
Workflow execution DAGs are read-only status views, but React Flow still exposes default keyboard and accessibility descriptions for editable graphs. Assistive snapshots can say nodes may be moved and edges may be deleted, even though DataBrew does not support editing execution DAG topology.

## What Changes

### New Capabilities
- None.

### Modified Capabilities
- Workflow execution DAG nodes and edges SHALL expose read-only semantics to keyboard and accessibility tooling.
- Workflow execution DAG SHALL keep operational interactions: node selection, node search, node action buttons, pan/zoom, and fit view.
- Workflow execution DAG dependency/status rendering SHALL remain unchanged.

## Impact
- **Affected code**: `Frontend/src/pages/WorkflowDagView.tsx`, related tests.
- **New APIs**: None.
- **Dependencies**: None planned.

## Scope
- **In scope**: Remove or override misleading React Flow move/delete accessibility descriptions on the execution DAG.
- **In scope**: Preserve existing mouse and keyboard access to run inspection controls.
- **In scope**: Verify the deployed dev UI with Chrome DevTools MCP.
- **Out of scope**: Pipeline designer canvas behavior, workflow execution behavior, DAG layout changes, backend/API changes, and persisted data changes.

## Success Criteria
- [ ] Chrome DevTools MCP snapshot no longer exposes move/delete instructions for execution DAG nodes or edges.
- [ ] Users can still search nodes and select a DAG node.
- [ ] Node action buttons for logs/runtime/io/terminal/detail remain available.
- [ ] Pan/zoom/fit controls continue to work.
- [ ] Dev Worker deployment is verified on the complex DAG regression URL.
