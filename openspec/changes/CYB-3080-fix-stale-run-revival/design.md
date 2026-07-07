# Design — CYB-3080

## Architecture Context
- 触发链路（dev 上真实复现两次）：批量任务子任务提交 → 资源守卫判定节点 CPU 请求（14 核）超过执行目标上限（8 核）→ run 正确标记 `Failed`，带清楚的中文错误信息（`RecordBatchSubtaskFailure` → `persistRunObservation`）。子任务 run 没有 Argo workflow（提交前就被拦下），也没有 `ArgoWorkflowUID`，workflow 名是占位名（含 `-batch-`）。
- 批次详情页 / 列表页读取 run 走 `GetRun`（`usecase.go:3987`），它依次调用 `refreshPipelineRunStatus` → `reconcileTerminalRunFromLedger` → `reconcileMisclassifiedRunFromArgo`。
- `reconcileMisclassifiedRunFromArgo`（`usecase.go:2287`）对 `Failed`/`Error` 的 run 会进入一段"可能只是 workflow 还没创建好，先等等"的宽容期复活逻辑：
  ```go
  if isPendingBatchWorkflowCreation(run) && shouldWaitForWorkflowCreation(run, now) {
      run.Status = "Pending"; run.Message = ""; run.FinishedAt = nil
      uc.persistRunObservation(ctx, run)
      return
  }
  ```
- 病根在 `isPendingBatchWorkflowCreation`（`usecase.go:5398`）：只要 run 有 BatchJobID、没有 UID、workflow 名是占位名（`isBatchSubtaskPlaceholderWorkflowName` = 名字含 `-batch-`），就返回 `true` —— **完全不看 `run.Status` 是不是已经 `Failed`**。于是一个被资源守卫确定性拒绝的 run，被当成"还在等创建"反复复活成 `Pending`、错误信息被清空，之后无限轮询一个永远不会存在的 workflow，表现为"静默卡住"。

## 为什么 PR #300 的修法是错的
PR #300 把守卫加在了 `refreshRunStatus`。但在 `GetRun` 路径里，`refreshRunStatus` 对一个（从 DB 新鲜读出的）`Failed` run 在函数第一行 `!isActiveDeploymentStatus(run.Status)` 就直接 return 了 —— 根本走不到那段守卫。真正复活它的是 `reconcileMisclassifiedRunFromArgo`。所以 PR #300 修的是一条对本 bug 不生效的路径，dev 上复现依旧失败。本次回退该改动。

## Goals
- 在**唯一的收敛点**修复：所有复活最终都经过 `persistRunObservation`，在那里用 DB 中的权威状态（而非调用方传入的内存对象）判定，一处覆盖全部路径。
- 不破坏 `reconcileMisclassifiedRunFromArgo` 既有的、有意为之的误判恢复（TTL 清理误判）能力。

## Non-Goals
- 不改资源守卫本身的判定逻辑或上限配置（那是模板/配额问题，见后续关于动态容量的讨论）。
- 不改 `PauseJob`/`StopRun` 失败时不重置 item 的 gap（独立问题，另行跟进）。

## Affected Modules
- `backend/internal/usecase/pipeline/usecase.go`：
  - `persistRunObservation`：新增一条与既有 `Succeeded` 单调性护栏并列的护栏。
  - 新增 `isDefinitiveTerminalFailure` 辅助函数。
  - 回退 PR #300 加在 `refreshRunStatus` 的 re-fetch 守卫。

## Architecture Decision

### Decision: 在 persistRunObservation 增加"确定性失败不可倒退"单调性护栏
- **Approach**：`persistRunObservation` 一进来就用 `runRepo.FindByID` 读到权威的 `existing`。既有护栏（CYB-3058）只保护 `existing.Status == Succeeded` 不被倒退成活跃态。新增并列的一条：若 `existing` 是"确定性终态失败"（`Failed`/`Error` 且带真实、非瞬时的错误信息）而传入的是活跃态（Pending/Running/…），则拒绝倒退，`*run = *existing` 后返回。
- **判定"确定性失败" vs "可恢复误判"**：用**消息**区分。
  - 确定性失败（资源拒绝等）：`Failed`/`Error` + 非空且**不是** `isStaleWorkflowUnavailableMessage` 的真实消息 → 不可复活。
  - 误判（TTL 清理假阳性、awaiting-deploy 占位）：带 `isStaleWorkflowUnavailableMessage` 认得的占位消息 → 仍可复活，行为不变。
- **Alternative**：改 `isPendingBatchWorkflowCreation` 让它对 `Failed` 返回 false。也能修，但它有 4 个调用点、语义是"是否在等创建"，改它需要逐一验证每个点的下游行为；而 `persistRunObservation` 是所有复活的唯一写入收敛点，改这一处最小、最集中、且天然免疫"调用方传入过时快照"（它读的是 DB）。
- **Rationale**：镜像已经存在且被充分理解的 `Succeeded` 护栏，认知负担低；一处修复覆盖 `reconcileMisclassifiedRunFromArgo` / `refreshRunStatus` / `markRunWorkflowNotFound` 等所有会调用 `persistRunObservation` 的路径。
- **Trade-off**：`persistRunObservation` 每次多一个廉价的内存判断（`existing` 本来就已经查出来了，不新增 DB 调用）。

## Pseudocode（核心改动）
```go
// persistRunObservation, after the existing Succeeded guard:
if isDefinitiveTerminalFailure(existing) && isActiveDeploymentStatus(run.Status) {
    slog.Warn("persistRunObservation: ignoring active-status regression on definitively-failed run", ...)
    *run = *existing
    return
}

func isDefinitiveTerminalFailure(run *models.PipelineRun) bool {
    if run == nil { return false }
    switch strings.ToLower(strings.TrimSpace(run.Status)) {
    case "failed", "error":
    default:
        return false
    }
    msg := strings.TrimSpace(run.Message)
    return msg != "" && !isStaleWorkflowUnavailableMessage(msg)
}
```

## Risks / Trade-offs
| Risk | Impact | Mitigation |
|---|---|---|
| 护栏过度拦截，挡住合法复活 | 误判恢复（TTL 清理假阳性）失效 | 用消息区分：占位/瞬时消息不算确定性失败，仍可复活；已有 `TestRefreshRunForList_RevivesRecentTTLNotFoundMisclassification` + 新增 `TestPersistRunObservation_MisclassifiedFailureStillRevivable` 双重锁定 |
| 真实运行时失败（带真实消息）被永久钉死 | 若 Argo 真的又把它跑起来则无法反映 | Argo 不会 un-fail 一个真实失败；真实失败本就该终态，需要重跑走新 run/retry，不靠倒退本 run 状态 |
