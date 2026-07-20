# Specs — Pipeline (CYB-3672)

## MODIFIED Requirement: 终态 misclassified run 的 Argo 再核实策略

**Priority**: P1

**Rationale**: 未加 age gate 时,已 TTL 清理的老 workflow 会被无限次 GetWorkflow,argo-server 稳定被 ~1 req/s 静默打入 ERROR 日志。既是资源浪费,也淹没真信号。

The system SHALL(revised):

- **R1** 对状态属于 `Failed / Error / Expired` 的 pipeline run,只有满足下述任一条件时才调用 Argo `GetWorkflow` 进行 revive 检查:
  - a) run 的创建时间(`CreatedAt` 或 `StartedAt`)距今 < staleActiveRunMaxAge(48h),**或**
  - b) run 的 `Message` 非空且非已知 stale marker(`staleWorkflowTTLCleanupMessage` / `messageWorkflowAwaitingDeploy` / `messageWorkflowUnavailable`),**或**
  - c) run 缺失时间戳(测试 fixture 场景)
- **R2** 不满足 R1 的 run(过 48h + 空消息或已知 stale marker + 时间戳非零),直接 early-return,不发起任何 Argo 请求。

### Scenario: 过期老 legacy 终态 run 不再打 Argo

**Given** 一个 pipeline_run,status=Error,message="",created_at 距今 60 天
**When** batch list refresh 触发 `reconcileMisclassifiedRunFromArgo`
**Then** SHALL NOT 调用 `wfClient.GetWorkflow`;run 状态和消息不变;argo-server 不再产生 not-found 错误日志

### Scenario: revival window 内的终态 run 仍然 re-check

**Given** 一个 pipeline_run,status=Error,message="Kubernetes 调度失败:...",created_at 距今 30 分钟
**When** `reconcileMisclassifiedRunFromArgo`
**Then** SHALL 正常调用 `wfClient.GetWorkflow`;若 Argo 返回 Running,run.Status 被 revive 为 Running(保留原 behavior)

### Scenario: 有真实诊断的老 run 仍然 re-check(safety)

**Given** 一个 pipeline_run,status=Error,message="invalid argument: 执行目标 ...",created_at 距今 90 天
**When** `reconcileMisclassifiedRunFromArgo`
**Then** SHALL 调用 `wfClient.GetWorkflow`(诊断消息保留 operator 的介入意图,不因 age 就静默跳过)
