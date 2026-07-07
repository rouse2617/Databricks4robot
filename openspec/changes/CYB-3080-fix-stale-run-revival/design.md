# Design — CYB-3080

## Architecture Context
- 触发链路（dev 上真实复现）：批量任务子任务提交 → 资源守卫判定节点 CPU 请求（14 核）超过执行目标上限（8 核）→ run 正确标记 `Failed`，带清楚的中文错误信息（`persistRunObservation` 记录）。
- 几乎同一时刻，批次列表页（`/runs?executionView=batch`）触发 `GET /api/v1/pipelines`/`GET /api/v1/backfill` 分页查询，逐个 run 走 `RefreshRunForList`（`usecase.go:2343`）→ `refreshRunStatus`（`usecase.go:2537`）。
- `RefreshRunForList` 收到的 `run` 参数来自这次列表查询自己的快照，可能是在资源守卫写入 `Failed` **之前**取到的（仍是 `Pending`）。
- `refreshRunStatus` 完全基于这个（可能过时的）`run.Status`/`run.Message` 做判断：`GetWorkflow` 返回 NotFound → `shouldWaitForWorkflowCreation` → `isPendingBatchWorkflowCreation`（末尾 `return strings.EqualFold(run.Status, "Pending")`，直接读内存字段）→ 判定"值得等待创建" → 清空 `run.Message` → `persistRunObservation(ctx, run)`，把 DB 里刚刚正确写入的 `Failed` + 清楚消息，覆盖成 `Pending` + 空消息。
- `persistRunObservation`（`usecase.go:2079`）本身第一步就会 `uc.runRepo.FindByID` 拿到当前最新的 `existing`，但现有的 monotonicity guard 只保护 `existing.Status` 是 `Succeeded` 的情况（`usecase.go:2099`），没有保护"当前已经是终态 Failed，传入的是过时 Pending"这种情况——这条"等待创建"路径的语义跟 `reconcileMisclassifiedRunFromArgo` 的"TTL 清理误判恢复"完全不同，不该共用同一个"Failed 可复活"的默认放行。

## Goals
- 消除这一类"过时内存快照覆盖掉刚产生的真实终态"的竞态,而不影响 `reconcileMisclassifiedRunFromArgo` 现有的、有意为之的误判恢复能力。

## Non-Goals
- 不改资源守卫本身的判定逻辑或上限配置。
- 不改 `PauseJob`/`StopRun` 失败时不重置 item 的 gap（独立问题）。
- 不为了这个修复引入通用的"所有终态一律不可复活"规则——`Failed`/`Error` 在别的场景下仍需要可复活（TTL 误判恢复）。

## Affected Modules
- `backend/internal/usecase/pipeline/usecase.go`：`refreshRunStatus` 的 `errors.Is(err, argo.ErrNotFound)` 分支。

## Architecture Decision

### Decision: `refreshRunStatus` 在"等待创建复活"前重新核对当前持久化状态
- **Approach**：进入 `shouldWaitForWorkflowCreation` 分支后、在真正调用 `persistRunObservation` 复活为 Pending 之前，用 `uc.runRepo.FindByID(ctx, run.ID)` 重新读一次当前状态。如果当前持久化状态已经不是"活跃/待创建"（例如已经是 `Failed`/`Error`/`Succeeded`），说明这份 run 在拿到手之后已经被别的路径终态化了，直接跳过复活、不清空消息、不覆盖，`return`。只有当前持久化状态确认仍然是活跃/`Pending` 时，才继续走原来的"等待创建"逻辑。
- **Alternative**：在 `persistRunObservation` 里扩大 monotonicity guard,让它对所有终态（不只是 `Succeeded`）都拒绝被"活跃状态"覆盖。
- **Rationale**：选择前者而不是后者，是因为 `persistRunObservation` 的 `Failed`/`Error` 可复活行为是 `reconcileMisclassifiedRunFromArgo` 依赖的既有语义（TTL 清理误判恢复,注释里写明是有意为之)，不能不分场景地关掉。这次的 bug 只出在"等待创建"这一条路径上，应该在这条路径自己的入口处做防护,不牵动 `persistRunObservation` 的通用契约。
- **Trade-off**：多一次 `FindByID` 调用（每次进入这个分支时）,但这个分支只在 NotFound 时才会走到,不是高频路径,可以接受。

## Pseudocode（改动范围）

```go
if errors.Is(err, argo.ErrNotFound) {
    if shouldWaitForWorkflowCreation(run, time.Now().UTC()) {
        if isPendingBatchWorkflowCreation(run) && isStaleWorkflowUnavailableMessage(run.Message) {
            // NEW: don't trust the possibly-stale `run` snapshot — confirm the
            // persisted state is still active before reviving it as Pending.
            if current, ferr := uc.runRepo.FindByID(ctx, run.ID); ferr == nil && current != nil &&
                !isActiveDeploymentStatus(current.Status) {
                slog.Info("refreshRunStatus: skip stale-revive, run already terminal",
                    "runID", run.ID, "persistedStatus", current.Status)
                return
            }
            run.Message = ""
            uc.persistRunObservation(ctx, run)
        }
        return
    }
    uc.markRunWorkflowNotFound(ctx, run)
}
```

## Risks / Trade-offs
| Risk | Impact | Mitigation |
|---|---|---|
| 额外一次 DB 读 | 轻微延迟增加 | 只发生在 NotFound + 判定需要等待创建这个窄分支,频率低 |
| 误判恢复场景被误伤 | `reconcileMisclassifiedRunFromArgo` 的合法复活被挡住 | 这次改动只影响"等待创建"这一条路径,不touch `reconcileMisclassifiedRunFromArgo` 自己的判断逻辑 |
