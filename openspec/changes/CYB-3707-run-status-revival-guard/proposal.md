# Proposal — CYB-3707

## Why

重试一个失败 step 后,run 会**卡死在错误的 Succeeded 状态**,重试按钮消失、`POST /runs/{id}/retry` 返回 `400 runtime retry only supports failed or errored runs`,失败的 step 再也点不动。

样本 `a45cc195-…`(video-proc-prod / cyber-clust,step `smpl-body-fit`)证据链:

- **Argo 权威**(`kubectl get workflow`):`phase=Failed`,`startedAt=2026-07-21T01:15:04`,`smpl-body-fit` 在 `01:15:04→01:23:35` 重跑并再次失败(exit 1)。startedAt 重置 + 节点重跑 = **一次重试在 01:15:04 真正生效**(顺带证明 pods-delete RBAC 修复已生效)。
- **DataBrew**:`status=Succeeded`,且 `startedAt(01:15) > finishedAt(07-20 10:10)`(不可能);资产节点账本仍正确记 `smpl-body-fit=Failed`。
- **后端日志**:`01:15:04.308 persistRunObservation status change Failed→Succeeded` + `01:15:04.351 GetRun status changed Failed→Succeeded` —— 一次**读**在重试重置 workflow 的同一刻把状态翻了。

**根因**:`GetRun` 每次读取都跑 3 个 reconciler([usecase.go:4799-4801](../../../backend/internal/usecase/pipeline/usecase.go))。`refreshPipelineRunStatusLive` 有 active-only 守卫([3119](../../../backend/internal/usecase/pipeline/usecase.go))、`reconcileTerminalRunFromLedger` 会从账本推出 Failed,唯一嫌疑是 `reconcileMisclassifiedRunFromArgo`([2869](../../../backend/internal/usecase/pipeline/usecase.go))→ `applyWorkflowToRun`([2944](../../../backend/internal/usecase/pipeline/usecase.go)):读到**重试重置中的瞬时 workflow** 派生出假的终态 `Succeeded`。`persistRunObservation` 只有 Succeeded/Failed→**active** 的守卫([2619](../../../backend/internal/usecase/pipeline/usecase.go)/[2636](../../../backend/internal/usecase/pipeline/usecase.go)),**没有 Failed→Succeeded 守卫**,假升级畅通持久化;之后 Succeeded 的 run 不再被任何 reconciler 重派生 + 单调守卫锁死,retry 门槛([5642](../../../backend/internal/usecase/pipeline/usecase.go) 仅 Failed/Error)就永久拒绝。属 CYB-3080 run 状态复活家族。

## What Changes

三处收敛在 run 状态派生/持久化,让状态始终正确 → retry 门槛与前端按钮**无需改动**即恢复:

### Modified Capabilities

1. **防误升(派生侧)** — `reconcileMisclassifiedRunFromArgo` / `applyWorkflowToRun`:当拉到的 workflow 处于 active/mid-retry(phase 非干净终态,或存在 Failed 叶子节点)时,**只置 Running(重观测),绝不派生终态 Succeeded**。
2. **守卫(持久化侧)** — `persistRunObservation` 增补:拒绝把 `Failed/Error` 的 run 覆盖为 `Succeeded`,**除非**该 Succeeded 来自一次权威的 live Argo 读、且 workflow 干净终态成功(Argo 从不 un-succeed;来自瞬时/非权威源的 Succeeded 盖 Failed 一律是 bug)。
3. **解锁已卡 run(对账侧)** — 当权威 live Argo 读显示 workflow **终态 Failed** 而 DataBrew 显示 Succeeded 时,把 DataBrew 纠回 Failed(唯一合法的 Succeeded→Failed,以 Argo 为准)。这会在下次读取时自动解锁 `a45cc195` 及同类 run。

状态保持正确 Failed 后:前端按钮(按 status 显示)自然出现 → 可点 → retry **只重跑失败 step**(Argo retry 语义,成功步保留)→ 因竞态不再误升 Succeeded,状态维持 Failed → **可反复点**。

## Impact

- **Affected code**: `backend/internal/usecase/pipeline/usecase.go`(`persistRunObservation`、`reconcileMisclassifiedRunFromArgo`、`applyWorkflowToRun`,及所需小 helper)
- **New APIs**: 无(响应 shape 不变)
- **Frontend**: 不改(重试按钮 keyed on status,状态修正后自然恢复)
- **Dependencies / migration**: 无

## Scope

- **In scope**: 防误升 + Failed→Succeeded 守卫 + 权威 Argo=Failed 的纠回;针对性单测(重试竞态、守卫、纠回)
- **Out of scope**: `smpl-body-fit` 的 exit-1 应用错误本身(retry 会重跑该步,修其失败是另一回事);跨集群定价(CYB-3705 已完成)

## Success Criteria

- [ ] `a45cc195-…` 在部署后下次读取时状态从 Succeeded 纠回 Failed,UI 出现重试按钮
- [ ] 点重试后 run 保持 Failed(不再自锁为 Succeeded),可反复重试
- [ ] 新增单测覆盖:mid-retry 读不派生 Succeeded、Failed→Succeeded 被守卫拦截、Argo 终态 Failed 纠回 Succeeded
- [ ] `go test ./internal/usecase/pipeline/...` 全绿
