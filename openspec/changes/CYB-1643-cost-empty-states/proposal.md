# Proposal — CYB-1643

## Why
Workflow detail currently uses one warning copy for different cost states, making pending nodes look like backend pricing failures.

## What Changes

### New Capabilities
- Workflow detail cost empty states will distinguish running/pending cost snapshots from unavailable cost configuration.
- Asset-node cost rows will use user-facing copy that matches the run/node state.

### Modified Capabilities
- The existing "暂无估算成本" warning will no longer be shown for nodes that have not produced a resource snapshot yet.

## Impact
- **Affected code**: `Frontend/src/pages/WorkflowDetailPage.tsx`, related workflow detail tests
- **New APIs**: None
- **Dependencies**: None

## Scope
- **In scope**: Frontend state classification and copy for workflow detail cost/asset-node panels.
- **Out of scope**: Backend cost calculation, pricing YAML changes, OpenCost/GCP Billing actual cost integration.

## Success Criteria
- [ ] Pending or Running nodes show copy indicating the cost snapshot will be generated after runtime resources are available.
- [ ] Completed nodes without cost show a quiet "no cost data" state instead of implying the backend pricing config is broken.
- [ ] Pricing/config unavailable copy is reserved for true unavailable cost summaries.
- [ ] Existing workflow detail data, logs, Pod diagnostics, and event timeline remain usable.
