# Tasks — CYB-3080

## Implementation
- [x] [backend] 回退 PR #300 加在 `refreshRunStatus` 的 re-fetch 守卫（targets 了一条对本 bug 不生效的路径）。
- [x] [backend] 新增 `isDefinitiveTerminalFailure`（`Failed`/`Error` 且带真实、非 stale-unavailable 消息）。
- [x] [backend] `persistRunObservation` 新增并列于 `Succeeded` 护栏的"确定性失败不可倒退成活跃态"单调性护栏。

## Local verification
- [x] `cd backend && go test ./internal/usecase/pipeline/...`
- [ ] `cd backend && go test ./...`
- [ ] `make fmt && make vet`

### 测试要点
- [x] `TestPersistRunObservation_DefinitiveFailureNotRegressedToActive`：Failed+真实消息 被 Pending 复活 → 拦截，保持 Failed+消息。
- [x] `TestReconcileMisclassifiedRunFromArgo_DoesNotReviveResourceRejectedRun`：复现真实路径（占位 `-batch-` 名 + 真实资源拒绝消息），保持 Failed。
- [x] `TestPersistRunObservation_MisclassifiedFailureStillRevivable`：stale-unavailable 消息的 Error run 仍可复活（护栏不过度拦截）。
- [x] 既有 `TestRefreshRunForList_RevivesRecentTTLNotFoundMisclassification` 保持通过（误判恢复端到端不受影响）。
- [x] 已用 `git stash` 验证：去掉 fix 后两个确定性失败测试失败（复现 bug），misclassification 测试仍通过。

## Deploy verification
- [ ] 部署 backend dev。
- [ ] 用同一个会被资源守卫拒绝的模板（`77b53ea3-...`，含 14 核节点）重新提交一个最小批次。
- [ ] 跨多个轮询周期（1–2 分钟）确认对应 run 稳定停在 `Failed` 且保留资源拒绝消息，不再被冲回 `Pending`。

## PR
- [ ] PR 描述包含 Linear ID（CYB-3080）与 OpenSpec change-id，说明这是对 PR #300 的纠正（改在 `persistRunObservation` 收敛点，回退 `refreshRunStatus` 误改）。
