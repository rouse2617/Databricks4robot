# Proposal — CYB-1697

## Why
Pipeline execution list node counts include Argo DAG/root nodes, so users see a larger count than the business step count shown on the execution detail page.

## What Changes

### New Capabilities
- None.

### Modified Capabilities
- Pipeline execution summaries SHALL display business step counts consistently across the execution list and execution detail pages.
- Live Argo workflow summaries SHALL exclude non-business DAG/root/controller nodes from the displayed node count.

## Impact
- **Affected code**: `backend/internal/handlers/workflow`, `Frontend/src/pages/WorkflowExecutionList.tsx` tests if needed.
- **New APIs**: None.
- **Dependencies**: None.

## Scope
- **In scope**: Normalize workflow summary `nodeCount` so execution list counts only business workflow steps.
- **In scope**: Add regression coverage for a workflow with one DAG root node and two Pod step nodes.
- **Out of scope**: Changing Argo workflow node detail payload shape, DAG rendering, cost summaries, or persisted pipeline run records.

## Success Criteria
- [ ] For `test-77705a`, execution list shows node count `2`, matching the execution detail business steps.
- [ ] Workflow summary counts exclude Argo DAG/root/controller nodes while retaining normal step nodes.
- [ ] Existing workflow detail DAG rendering still receives the full node list needed to render DAG and timeline state.

## Goals (SLO)
- **Latency**: No measurable extra API latency; count is derived from already-loaded workflow status nodes.
- **Concurrency**: No change.
- **Quality**: Backend regression test covers DAG-root exclusion.
