# Proposal — CYB-1649

## Why
线上 dev 环境 5 个 pipeline 执行记录卡在 "Running" 状态数小时不终止：4 个 Pod Pending/Unschedulable（请求资源超出集群容量），1 个 Pod ImagePullBackOff（smoke 测试故意用不存在镜像）。根因是 transpiler 未设默认 `ActiveDeadlineSeconds`，Argo 永久保持 Running；后端 `refreshRunStatus` 只镜像 Argo 状态不检测长时间无进展的 workflow。

## What Changes

### Part A — transpiler 默认超时
- 新增 `DefaultActiveDeadlineSeconds = 7200` (2h) 常量
- `Transpile()` 中当 opts.ActiveDeadlineSeconds == 0 时自动应用默认值

### Part B — refreshRunStatus staleness 检测
- 新增 `detectStuckWorkflow` 辅助函数：检查 Pod 节点是否长时间 Pending 且消息包含 Unschedulable/ImagePullBackOff/CrashLoopBackOff
- 检测阈值：pod stuck > 30min 或 workflow 整体 > 6h
- 检测到卡住时自动标记为 "Error"，写入原因 message
- `UpdateStatus` 加 `message` 参数持久化错误原因

## Impact
- **Affected code**: `backend/internal/transpiler/transpiler.go`, `backend/internal/usecase/pipeline/usecase.go`, `backend/internal/repository/pipeline_repository.go`, `backend/internal/postgres/pipeline_repo.go`
- **No API surface change**
- **No migration needed** (message 列已存在)
- **No frontend change**

## Scope
- **In scope**: transpiler 默认超时 + 后端 staleness 检测
- **Out of scope**: 可配置超时（per-template/per-component）、前端超时 UI

## Success Criteria
- [ ] transpiler 不加 `ActiveDeadlineSeconds` 时自动应用 7200s
- [ ] Pod Unschedulable > 30min → run 自动标记 Error
- [ ] workflow Running > 6h → run 自动标记 Error
- [ ] 正常 Running 的 workflow 不受影响
- [ ] ErrNotFound → Expired 行为不变
