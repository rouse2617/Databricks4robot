# Proposal — CYB-1646

## Why
Pipeline workflow operations screens currently collapse several different runtime states into generic empty copy. During browser regression this made completed runs without cost snapshots, pending nodes, and troubleshooting surfaces harder to understand.

## What Changes

### New Capabilities
- None.

### Modified Capabilities
- Workflow execution details SHALL distinguish missing monitoring and cost snapshots by runtime state instead of repeating broad "no data" copy.
- Node troubleshooting surfaces SHALL use product-facing Chinese copy for unavailable logs and pending node states.
- Pipeline run entry points SHALL make no-asset runs and version selection clearer.

## Impact
- **Affected code**: `Frontend/src/pages/WorkflowDetailPage.tsx`, `Frontend/src/components/pipeline/WorkflowNodeDetailPanel.tsx`, `Frontend/src/pages/PipelinePage.tsx`, `Frontend/src/components/pipeline/DeployPanel.tsx`, related tests.
- **New APIs**: None.
- **Dependencies**: None planned.

## Scope
- **In scope**: Cost and monitoring empty-state copy on workflow execution detail and node detail.
- **In scope**: Pending or unavailable log copy in node troubleshooting surfaces.
- **In scope**: Run dialog/button wording for no-asset runs and version selection.
- **In scope**: Focused frontend tests plus Chrome DevTools MCP verification on dev.
- **Out of scope**: Backend workflow execution behavior, API contracts, persisted workflow data, and Kubernetes/Argo scheduling logic changes.
- **Out of scope**: Large visual redesign of DAG layout, component marketplace metadata model, and asset batch-run navigation IA.

## Success Criteria
- [ ] Completed workflow runs without a cost snapshot say the snapshot is not generated, not repeated generic "暂无成本数据".
- [ ] Pending workflow cost fields say "等待资源" where cost cannot exist yet.
- [ ] Pending or unavailable node log states use Chinese product copy and do not expose raw English technical text.
- [ ] Run entry points use "运行" for no-asset runs and show that the run is not bound to assets.
- [ ] Chrome DevTools MCP verifies the deployed dev UI on workflow detail and run dialog paths with no new console errors.
