# Proposal — CYB-1531

## Why
Pipeline runs can execute successfully in Argo, but users cannot reliably inspect logs or trust drag-and-drop component selection in the Databrew Pipeline UI.

## What Changes

### New Capabilities
- Pipeline workflow detail shows logs for the real Kubernetes pod backing a selected Argo node.
- Pipeline designer preserves the exact component selected by drag-and-drop.

### Modified Capabilities
- Workflow node details distinguish Argo node identity from Kubernetes pod identity.
- Pipeline run views reduce operator confusion around sparse DAG layouts and low-level execution metadata where feasible.

## Impact
- **Affected code**: `backend/internal/argo`, `backend/internal/handlers/workflow`, `Frontend/src/pages`, `Frontend/src/components/pipeline`, `Frontend/src/api`
- **New APIs**: None expected; existing workflow detail/log response shape may be extended with pod name metadata.
- **Dependencies**: None expected.

## Scope
- **In scope**:
  - Resolve real pod names for workflow Pod nodes before requesting logs.
  - Display real pod name in node details.
  - Fix component drag/drop identity so the dropped node matches the dragged component.
  - Add targeted backend/frontend tests where the local code structure supports it.
  - Deploy frontend/backend dev as needed and verify with Chrome DevTools MCP.
- **Out of scope**:
  - New Pipeline Run database model.
  - New batch/grouping backend APIs.
  - Full execution-target management UI.
  - Mobile layout.

## Success Criteria
- [ ] A succeeded workflow Pod node shows non-empty logs in the UI when Kubernetes logs exist.
- [ ] The node detail panel shows the real Kubernetes pod name, not only the Argo node id.
- [ ] Dragging `codex-valid-emit-*` onto the canvas creates an `emit` node, and dragging `codex-valid-transform-*` creates a `transform` node.
- [ ] Existing pipeline deploy/list/detail behavior remains intact.
- [ ] Dev deployment is verified against the Cloud Run frontend URL with Chrome DevTools MCP.
