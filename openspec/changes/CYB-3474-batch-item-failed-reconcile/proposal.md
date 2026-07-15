# CYB-3474 — batch 子任务 item 卡在 failed 无法自愈

## Problem

批量任务 `nw-delivery-dev-1784048739953`（job `d655b177-af99-4f1f-ad7e-f2394bae5d2c`，335 子任务，dev）出现大量"假失败"：DataBrew 账本里 item 持久状态是 `failed`，但底层 Argo workflow 实际 `Succeeded`。

同一时刻两个独立权威源实测：

- Argo 真实 phase（kubectl，batch 的 uuid workflow）：**29 Failed** / 218 Succeeded / 76 Running
- `SummarizeItemStatuses`（纯 SQL count，node-summary 强制 sync 后）：**91 failed**

→ **≥62 个 item 被持久化写成 `failed`，但其 Argo workflow 并未失败**。数据其实已交付（workflow Succeeded），是账本错、不是数据错。后果：batch `failedCount` 虚高约 3 倍，且这些 item 是终态、不会自动重试；批次最终会以 `failed` 收尾而非 `completed`。

## Root cause

自愈通路**本就存在**：`ListRunSummaries(RefreshActive:true)` → `refreshRunSummariesForList` 的 **Pass 1** 会对误判的 `Failed/Error/Expired` run 调 `RefreshRunForList` → `reconcileMisclassifiedRunFromArgo` → 命中 Argo 真值后 `applyWorkflowToRun` + `persistRunObservation` → `syncBackfillItemStatusFromRun` 回写 item 到 `completed`。

缺的是**覆盖**，两处限制导致大量 failed run 从没被 reconcile 到：

1. `syncJobProgressInternal`（backend/internal/usecase/backfill/usecase.go:1487）只捞 `status IN ('running','pending')` 的 item；`fetchBatchRunsByIDMap` 随后 `ListRunSummaries(PageSize = len(running/pending items))` —— failed run 落在这一页之外就不进 reconcile 集合。
2. `refreshRunSummariesForList` 有 `maxActiveDeploymentStatusRefresh = 50` 上限，单次最多 reconcile 50 个。

reaper（`ResetStaleItems`）只碰 `running`，也碰不到 failed item。于是被误标的 failed item 永久卡住；只有按需 `GetRun`（详情页）能临时算对，但列表/计数读的是 raw item，仍错。

（item 被首次写成 failed 的确切触发点为推断：parallelism=20 churn 期，workflow 长期卡 Argo Pending，某次读取期 reconcile 出瞬时 Failed 落库。本修复为自愈收敛，**不依赖**精确复现该写入。）

## Scope

- `syncJobProgressInternal` 增加一个**有界的 failed-item reconcile pass**：捞该 job 的 `failed` item（上限 N，默认 50/次），对每个的 run 走既有 reconcile（`reconcileMisclassifiedRunFromArgo` 路径），命中 Argo 非失败真值时经既有 `syncBackfillItemStatusFromRun` 回写 item。大批次跨多次 sync 收敛，避免一次性打爆 Argo API。
- 单测覆盖：failed 且 Argo=Succeeded → 纠回 completed；failed 且 Argo=Failed → 保持 failed；有界性（超过 N 不一次全捞）。

## Out of scope

- 28 个"孤儿"子任务（半提交、从不在 Argo）：另行 `POST /api/v1/backfill/{id}/resume` 处理，无需 SQL、无需本次改码。
- Argo controller `parallelism`（20→150，已在运行时处置；helm values 同步用户另行决定）。
- 前端列表状态陈旧 / 计数三处打架 / "运行中"应为"等待中"：独立前端问题，另开。
- item 首次被误写 failed 的**预防**（区分瞬时 vs 真失败）：作为后续硬化项，非本次。

## Alternatives considered

- **抬高/去掉 `maxActiveDeploymentStatusRefresh`**：拒绝，无界 Argo 负载。
- **`fetchBatchRunsByIDMap` 改为分页拉全部 run**：仍受 50 cap 限制，且放大每次 list 开销。
- **扩 `syncJobProgress` 工作集直接含 'failed'**：可行但需确保 failed run 真被 reconcile（RefreshActive 默认走 Pass1，但受同样 cap/分页限制）；显式有界 pass 更可控。
