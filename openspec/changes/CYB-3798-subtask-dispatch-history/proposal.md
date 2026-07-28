# CYB-3798: 订阅任务 — 历史下发批次与资产下钻

## Problem

CYB-3778 的订阅任务是「发了就不管」的下发器：收到 Pub/Sub 消息 → fan-out → 每个绑定建一个 batch。但订阅任务只持久化 `last_batch_ids`（最近一次下发），UI 无法从订阅任务点进去看它**历史下发了哪些批次、每批跑过哪些资产**。

数据其实都在，只是没有查询入口 + 前端下钻：
- `backfill_jobs.created_by = "subscription-task:<任务id>"` 反查订阅任务的全部批次；
- `backfill_items.asset_id`（+ 每个 asset 的状态/run）记录每批处理的资产。

## Solution

新增两个**只读**接口 + 前端下钻。无 migration、不改鉴权、不碰 off-limits：

1. `GET /api/v1/backfill` 增加可选 `createdBy` 查询参数 → 反查某订阅任务的全部历史批次（响应形状不变，仍是 `{items: BackfillJob[]}`，仅加过滤）。
2. 新增 `GET /api/v1/backfill/:id/items` → 返回批次的资产明细 `{items: BackfillItem[]}`（`assetId` + `status` + `pipelineRunId` 等），复用已有的 `FindItemsByJobID`。

前端订阅任务面板：点任务 → 详情抽屉「历史下发批次」列表（按 `createdBy=subscription-task:<id>` 拉）→ 展开某批次 → 资产列表（asset + 状态），并链接到已有 batch / node-summary 视图。

## Key Decisions

1. **通用 `createdBy` 过滤**，而非专用 `/subscription-tasks/:id/batches` 接口 —— 最小改动、可复用；`"subscription-task:<id>"` 拼接约定封装在前端 `subscriptionTaskApi` 一个函数里，不散落。
2. **复用 `FindItemsByJobID`** —— 不新增表/列/migration，资产数据已在 `backfill_items`。
3. **`ListJobs` 改为接收 filter 结构**（`ListJobsFilter{CreatedBy *string}`）—— 单方法扩展、向后兼容；同一 commit 更新所有实现与 test mock（interface 变更规则）。
4. **SDK 暂不覆盖** —— `backfill` 是内部 admin 面（`X-Databrew-Token`），非公共 SDK REST 面；契约同步仍做 OpenAPI + api-guide + smoke + spec（见 `decisions.md`）。

## Scope

- **Backend**：`backfill` handler（list 加 `createdBy` param + 新 `/:id/items` 路由）、usecase（`ListJobs(filter)` + `ListItems(jobID)`）、repo（`WHERE created_by`、复用 `FindItemsByJobID`）、更新所有 mock。
- **Contract**：`api/openapi.yaml`、`docs/review/api-guide.md`、`scripts/smoke-subscription-tasks-dev.sh`、`specs/backfill/spec.md`。
- **Frontend**：`subscriptionTaskApi.ts`（`listBatches` / `listBatchItems`）+ `SubscriptionTasksPanel.tsx` 详情抽屉。
- **不做**：migration、auth 改动、后端分页（首版一次性拉，历史批次数量可控；量大再迭代）。
