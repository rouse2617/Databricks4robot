# Ralph 循环策略 v2

## 背景
iter 1-22 静态扫 TODO 已穿(主线代码干净)。真正信号来自外部事件:review-req / CI failure / Cloud Run ERROR / stranded workflow。
iter 22.6 用户手动介入合入 4 stalled PR(#419/#476/#403/#343)证明:仓库未启 auto-merge + 无默认 reviewer → Ralph 无法独立推 PR 到 merged,必须外部 human。

## DISCOVER 新顺序(硬优先级,替代原有子系统轮换)
每轮 DISCOVER 首跑 `.agent/loop/scripts/discover-signals.sh`,按段处理:
1. 段 1 review-request(非 self) → comment / approve 别人 PR
2. 段 4/5 backend ERROR / stranded → 记 P0 backlog(不自动改代码,先诊断)
3. 段 2 主线 CI failure → 我的 PR 则修;别人的记 P1 提醒
4. 段 3 我 stalled PR → `gh pr view --json reviewDecision,mergeStateStatus` 校准;>24h 无变化 → park + DEFERRED-HUMAN
5. 全空 → 跑 T1-T4+T7 baseline VERIFY,记 LESSON;仍空 → 静态扫兜底(60min 内已扫过则 skip)

## 已存在 PR 校准(L-M7 硬规则)
接手 IN_FLIGHT_PR≠None 时,**必须**先 `gh pr view <N> --json baseRefName,reviewDecision,mergeStateStatus,statusCheckRollup` 校准。
- base≠dev → `gh pr edit --base dev`
- reviewDecision="" + mergeState=CLEAN + CI 全绿 → 尝试 `gh pr merge --squash`;若报仓库未启 auto-merge → park + DEFERRED-HUMAN

## 元项 vs 业务项(L-M4)
- **元项**(改 `.agent/loop/`):无 PR,直接 DONE_ITEM 计 done_count
- **业务项**(改仓库源码):必走 CODING→AWAIT_CI→AWAIT_DEPLOY→VERIFY→DONE_ITEM

## Auto-merge 缺失应对(L-M8/新)
`gh pr merge --auto` 会报 `Pull request Auto merge is not allowed for this repository`。策略:
- CI 全绿 + reviewDecision="" → 直接 `gh pr merge --squash` 试;失败则 park
- 不再期待自己 approve 通过分支保护
