# Proposal — CYB-1703

## Why
Workflow execution detail shows component names on DAG cards but still shows Argo technical step ids in the node detail table, forcing users to mentally map `step-step-*` back to the pipeline component they configured.

## What Changes

### New Capabilities
- None.

### Modified Capabilities
- Workflow execution detail SHALL use business/component display names in the node detail table.
- Technical Argo or pipeline node ids SHALL remain available as secondary diagnostic metadata when useful.

## Impact
- **Affected code**: `Frontend/src/pages/WorkflowDetailPage.tsx` and related tests.
- **New APIs**: None.
- **Dependencies**: None.

## Scope
- **In scope**: Resolve node detail table labels from live workflow/pipeline snapshot display names where available.
- **In scope**: Keep status, logs, Pod diagnostics, and cost cells unchanged.
- **Out of scope**: Backend ledger migration, Argo node naming changes, DAG card layout changes, or persisted run-node schema changes.

## Success Criteria
- [ ] On `test-77705a`, node detail rows display `Count Lines` instead of `step-step-2` / `step-step-3`.
- [ ] Users can still inspect the technical node id via secondary text or tooltip.
- [ ] DAG cards, node detail actions, logs, Pod diagnostics, and timeline interactions continue to work.

## Goals (SLO)
- **Latency**: No additional network calls; labels are derived from already-loaded workflow/run state.
- **Concurrency**: No change.
- **Quality**: Targeted frontend test covers table display-name mapping.
