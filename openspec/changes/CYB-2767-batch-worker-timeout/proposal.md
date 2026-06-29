# Proposal — CYB-2767

## Why

批量任务的 worker goroutine 在没有超时的 context 下调 Argo API，如果 Argo 暂时无响应，worker 永远不返回，整个 batch 永久 stuck 在 "running" 状态。

## What Changes

### Modified Capabilities

- **Batch worker timeout**: `processBatchJob`（pipeline batch.go）和 `runItems`（backfill usecase.go）中的每个 item 处理调用增加 context timeout，超时后该 item 标记失败而不是永久阻塞。
- **Backfill item execution**: `executeItem` 中的 `DeployByTemplateID` 调用没有超时保护，加入 per-call timeout。

## Impact

- **Affected code**: `backend/internal/usecase/pipeline/batch.go`, `backend/internal/usecase/backfill/usecase.go`
- **New APIs**: 无
- **Dependencies**: 无新增依赖

## Scope

- **In scope**: batch job worker 中 `CreateRunByTemplateID` / `executeItem` 的 per-item timeout
- **Out of scope**: 不涉及 Argo 侧的超时配置；不涉及前端显示；不涉及 DB schema 变更

## Success Criteria

- [ ] 当一个 Argo API 调用 hang 住超过 timeout（60s），该 item 标记为 failed，其他 items 不受影响
- [ ] 所有 items 处理完毕后 batch 能正确进入 terminal 状态（completed / failed）
