# Proposal — CYB-1644

## Why
Workflow node monitoring and billing panels currently collapse several different runtime states into the same "暂无监控数据 / 暂无计费数据" warnings, which makes Pending or Unschedulable nodes look misconfigured instead of simply not ready yet.

## What Changes

### New Capabilities
- Workflow node monitoring empty states will distinguish waiting-for-runtime metrics from true no-snapshot or unavailable cases.
- Workflow node billing empty states will distinguish waiting-for-cost-snapshot from completed-without-cost and unavailable cases.
- Workflow detail log viewer will use user-facing copy when live Argo logs cannot provide historical pagination or when a node has not started producing logs yet.

### Modified Capabilities
- The runtime tab will no longer show the same generic monitoring/billing warning for Pending, Running, and completed nodes.
- The runtime tab terminal guidance will be condensed so it does not repeat the same instruction as the node card affordance.

## Impact
- **Affected code**: `Frontend/src/components/pipeline/WorkflowNodeDetailPanel.tsx`, `Frontend/src/components/pipeline/WorkflowNodeDetailPanel.test.tsx`, `Frontend/src/pages/WorkflowDetailPage.tsx`, related workflow detail tests
- **New APIs**: None
- **Dependencies**: None

## Scope
- **In scope**: Frontend state classification and copy for workflow node monitoring, billing, runtime-tab diagnostics, and related workflow detail log-viewer empty states.
- **Out of scope**: Backend metrics ingestion, pricing YAML changes, new monitoring or billing APIs, terminal exec backend, scheduler/RBAC changes.

## Success Criteria
- [ ] Pending or Running nodes show waiting-state copy for monitoring and billing instead of generic unavailable warnings.
- [ ] Completed nodes without monitoring snapshots or cost data show quiet no-data states that do not imply configuration failure.
- [ ] Unschedulable / not-started nodes explain the lack of logs in user-facing language when the log viewer opens.
- [ ] Runtime diagnostics, logs, Pod details, and terminal entry points remain usable after the copy/state changes.
