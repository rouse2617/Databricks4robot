# Decisions — CYB-3707

## 2026-07-21 — 用"成功必经 Running 中转"不变量做守卫,而非新增权威标记

- **Context**: 需要在 `persistRunObservation` 阻止 Failed→Succeeded 的误升,但不能误伤合法的 retry 成功,也不想大改观测链路去传"权威性"标记。
- **Decision**: 利用 Argo 不变量 —— retry/resubmit 一定先把 workflow 重置成 `Running` 才可能到 `Succeeded`。因此合法成功必然是 `Failed→Running→Succeeded`,**直接** `Failed→Succeeded`(中途未 active)必是瞬时误读,直接拒绝。
- **Alternatives**:
  - 给观测加"authoritative"标记透传到 persistRunObservation —— 侵入性大,被本不变量替代。
  - 把 retry 门槛改成看 live Argo 可重试性(不看 run.Status)—— 不修 UI 显示的错误 Succeeded、需改前端、不治根。作为 fallback 记录但不采用。
  - 手工改 DB 把 `a45cc195` 状态改回 Failed —— 只治单点、需手动数据变更;用对账(#3)通用化替代。
- **Rationale**: 自包含、零新增字段、不误伤合法路径,且与既有 CYB-3058/CYB-3080 守卫正交(只新增 Failed→Succeeded 一条,不动 →active 的两条)。

## 2026-07-21 — 恢复走 ledger 对账,不做 DB 手术

- **Decision**: 已卡死的 Succeeded run 通过在单 run 读路径检测"Succeeded 但账本有 Failed 叶子",复用 `reconcileTerminalRunFromLedger` 从账本推回 Failed。`Succeeded→Failed`(终态→终态)本就未被守卫拦截,自然生效。
- **Rationale**: 通用(所有同类 run 下次读取自愈)、无 migration、无手工改数据,符合"schema/数据变更走正规路径"原则。

## 2026-07-21 — OpenSpec checkpoint 批准

用户确认「OpenSpec OK，继续」,进入编码。

## 2026-07-21 — 编码期精化:精确机制 + 放弃 persistRunObservation 守卫(避免回归)

编码时定位到**精确机制**,并据此把 3 处改动收敛为 **2 处(更准更安全)**:

- **精确根因**:`applyWorkflowToRun` 的 CYB-3491 finalize-early 块([usecase.go:2366](../../../backend/internal/usecase/pipeline/usecase.go))在 workflow 仍 active 时用 `allBusinessPodsSucceeded(wf)` 提前 finalize。argo-server `/retry` 会**从 `wf.Status.Nodes` 删掉失败节点**让 controller 重建,那一刻只剩已成功的业务 pod → `allBusinessPodsSucceeded` 返回 true → 误升 Succeeded。这就是 01:15 那次翻转。

- **放弃**原设计的「`persistRunObservation` 拒绝直接 `Failed→Succeeded`」守卫:它与既有 `isDefinitiveTerminalFailure(existing) → active` 守卫(2636)冲突 —— 带真实错误消息(如 `main: Error (exit code 1)`)的 Failed 是 definitive,重试时的中间态 `Failed→Running` 会被 2636 挡下,于是"重试后真成功"最终以 `Failed→Succeeded`(中途未 active)到达,新守卫会**误杀合法成功**。故不加此守卫。

- **改为源头堵**(Change 1):finalize-early 增加完整性判据 —— 业务 step pod 数须 ≥ `run.NodeCount`(= `len(pipe.Nodes)`,提交时定死的步骤数)。retry 删掉一步 → pod 数 < NodeCount → 不 finalize,状态保持(active 被 2636 挡回 Failed)。既不误升,也不误杀合法成功。跳过步骤(Skipped 无 Pod 节点)时保守地延迟 finalize,属正确性安全的性能取舍;`NodeCount<=0` 的老 run 回退旧行为,由恢复兜底。

- **保留**恢复(Change 3):已卡死的 Succeeded run 通过单 run 读路径检测"Succeeded 但账本有 Failed 叶子"→ 复用 `inferTerminalRunFromAssetNodes` 推回 Failed(`Succeeded→Failed` 终态→终态,未被任何守卫拦截)。

最终 2 处代码改动:`applyWorkflowToRun` finalize-early 完整性判据 + `GetRun` 挂载 `recoverStuckSucceededRun`。

## 2026-07-21 — 部署验证走 CI(用户指令),推迟到合并后

- **Context**: deploy-before-commit 默认要求预提交手动部署 dev 验证。
- **Decision**: 用户明确指示「直接 pr 到 dev,然后 merge 走 cicd 部署」—— 跳过预提交手动部署(`local-build-deploy.sh`),改走 PR→merge→CI 自动部署,合并后验证。
- **Rationale**: 符合项目"dev 部署优先走 CI、少踩手动脚本坑"的经验;改动 Tier L 全绿 + 5 个针对性单测(含对真实机制的回归),风险可控;backend-only、无 migration。Rule precedence #1(用户显式指令)。
- **合并后验证点**: 读 `a45cc195-…` → 状态从 Succeeded 纠回 Failed、重试按钮恢复;点重试 → smpl-body-fit 重跑、run 保持 Failed(不自锁)。

## 2026-07-21 — 验证 Tier L

- **Decision**: 状态派生是跨切面核心逻辑,PR 前跑 `go test ./...`(Tier L),部署后对 `a45cc195` 做行为断言(状态纠回 + 重试按钮 + 可反复重试)。
- **Note**: 不改 Frontend(重试按钮 keyed on status,状态修正后自然恢复),故不触发 Chrome DevTools MCP 要求;但部署验证仍会在 UI 上肉眼确认按钮恢复。

## 2026-07-21 — 关联事实

- 根因验证同时**证明 pods-delete RBAC 修复已生效**:01:15:04 的重试成功删除旧 pod 并重跑 `smpl-body-fit`(见 proposal 证据)。RBAC 一事闭环。
- `smpl-body-fit` 的 exit-1 是应用层错误,本 change 不负责修;retry 只保证"能重跑失败步"。
