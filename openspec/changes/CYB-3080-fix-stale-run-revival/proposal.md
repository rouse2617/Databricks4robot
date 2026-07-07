# Proposal — CYB-3080

## Why

批量任务的子任务被资源守卫正确拒绝（`Failed`，带清楚的中文错误信息），但随后被一段基于过时内存快照的"等待 workflow 创建"逻辑覆盖回 `Pending` + 空消息，导致 run 无限期轮询一个永远不会存在的 Argo workflow，看起来像"静默卡住"而不是"清楚地失败"。

## What Changes

### Modified Capabilities
- `refreshRunStatus` 在决定"是否应该再等一等 workflow 创建"之前，SHALL 重新读取该 run 当前的持久化状态，而不是只信任调用方传入的（可能过时的）内存对象；如果当前持久化状态已经是终态（非 `Succeeded` 的场景下同样适用，例如资源守卫刚判定的 `Failed`），SHALL 跳过"等待创建"的复活逻辑。

## Impact
- **Affected code**: `backend/internal/usecase/pipeline/usecase.go`（`refreshRunStatus`；可能连带 `isPendingBatchWorkflowCreation`/`shouldWaitForWorkflowCreation` 的调用点)
- **New APIs**: 无
- **Dependencies**: 无新增依赖
- **Schema**: 无迁移

## Scope
- **In scope**:
  - `refreshRunStatus` 在"NotFound → 等待创建 → 复活为 Pending"这条路径上，覆盖前重新核对当前持久化状态。
  - 保持 `persistRunObservation` 现有的 `reconcileMisclassifiedRunFromArgo`（TTL 清理误判恢复）复活语义不变——这次的问题是另一条独立路径（等待创建宽容期）在错误场景下复活，不是要收紧误判恢复本身。
- **Out of scope**：
  - `PauseJob` 在 `StopRun` 失败时不重置 item 的 gap（已知的另一个独立问题，单独跟进）。
  - 资源守卫本身的规格/上限调整（这是模板配置问题，不是代码 bug）。

## Success Criteria
- [ ] 一个被资源守卫在提交阶段判定为 `Failed` 的 run，在后续任何"等待创建"轮询/列表刷新中，都不会被覆盖回 `Pending`。
- [ ] `reconcileMisclassifiedRunFromArgo` 的既有误判恢复行为不受影响（回归测试覆盖）。
