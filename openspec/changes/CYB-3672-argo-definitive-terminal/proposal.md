# Proposal — CYB-3672

## Why
`reconcileMisclassifiedRunFromArgo` 对所有终态(Failed/Error/Expired)misclassified run 每次列表刷新都调用 Argo `GetWorkflow`。当 Argo 已经 TTL 清理了对应 workflow(默认 30d),再问就永远 404 — "not reviving" 分支只 log 不改状态 → 下一轮 `needsMisclassifiedReconcile` 仍返回 true → 无限死循环。

生产观测(2026-07-18):共享 `cyber-databrew-dev/argo-server` 稳定 ~1 req/s 因此打 ERROR 日志,主要来源是历史积压的 `youxin-*`(1394 条,终态但 legacy)。前端拉列表越多,argo-server 越忙。

## What Changes

### Modified Capabilities
- `reconcileMisclassifiedRunFromArgo` 在**过 revival window(48h)+ message 无实际诊断**时,不再调用 `GetWorkflow`。空消息或已知 stale marker 都判定为 "无 diagnostic",直接 early-return。
- 有真实诊断消息(如 "Kubernetes 调度失败","invalid argument: 执行目标..." 等)的终态 run **保留原有行为**,仍然会 re-check Argo — operator 可能想主动验证一次。

## Impact
- **Affected code**: `backend/internal/usecase/pipeline/usecase.go`(`reconcileMisclassifiedRunFromArgo` + 新 helper `isNoisyMisclassifiedMessage` / `runHasCreatedTimestamp`)
- **测试**: 2 条新单测覆盖过期+跳过 与 revival window 内+调用 argo 的对偶行为
- **无 schema 变更,无 API 变更,无 migration**

## Scope
- **In**:仅 `reconcileMisclassifiedRunFromArgo` 的早退逻辑
- **Out**:不改 `needsMisclassifiedReconcile`(外层 gate);不改 `shouldReviveMisclassifiedWorkflowNotFound`(仍以 age 判 revive 与否);不做批量 SQL 清理历史死运行(数据侧,不在本变更内)

## Success Criteria
- [x] 单测:`TestReconcileMisclassified_SkipsArgo_WhenPastRevivalAge` + `TestReconcileMisclassified_CallsArgo_WithinRevivalAge` 双向覆盖 age gate
- [x] 全量 `go test ./...` 绿
- [ ] dev 部署后:观察 argo-server ERROR 日志速率显著下降(baseline ~60/min → 期望 < 10/min)
