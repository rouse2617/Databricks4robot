# Proposal — CYB-3068

## Why
`useWorkflowDetail.ts` 里两个独立的 `useEffect`(`loadWorkflow` 的挂载 effect、`loadRunEvents` 自己的挂载 effect)在组件挂载时都会触发同一份 run ledger 数据加载(`listRunEvents`/`listRunAssetNodes`/`getRunCostSummary`/`listRunInputs`/`listRunOutputs`/`getRunRuntime`,共 6 个并发请求)。两者之间没有互斥机制,导致每次挂载、每次手动刷新、每次轮询 tick 都把这 6 个请求各打两次,纯粹浪费后端/DB 负载。

## What Changes

### Modified Capabilities
- pipeline: WorkflowDetailPage 加载 run ledger 数据时不再重复请求

## Impact
- **Affected code**: `Frontend/src/pages/useWorkflowDetail.ts`
- **New APIs**: 无
- **Dependencies**: 无

## Scope
- **In scope**: `useWorkflowDetail` hook 内部的请求去重(挂载、手动刷新、轮询三个场景共用同一套触发路径)
- **Out of scope**:
  - 后端 singleflight 或其他跨请求合并机制(根因在前端,不需要后端配合)
  - `runId` 模式的行为(该模式已经通过同一个标志正确去重,不受影响)

## Success Criteria
- [ ] `workflowName` 模式下挂载一次,run ledger 的 6 个子资源请求各只发起 1 次(不是 2 次)
- [ ] 手动刷新按钮、活跃 run 的轮询场景同样只触发 1 次,不产生新的行为回归
- [ ] `getWorkflow` 失败时(无论 404 还是其他错误),ledger 数据仍会尝试加载且失败信息会正确出现在 `runEventState`/`assetNodeState`/`costSummaryState`/`runMetadataState` 里
