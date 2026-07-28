# CYB-3801: 订阅消息单条→单 run，多条→批次；下发历史合并展示

## Problem

CYB-3778 的订阅消费当前：每条消息只带一个 `asset_id`，且同一次 pull 的多条消息被合并成**一个批次**。发布方的真实场景是——有时推**一个**资产，有时推**一批**资产。用户希望据此区分下发形态：一个 → 单个 pipeline run（轻量，进「执行记录」）；一批 → 批次（`backfill_job`，进「批量任务」）。

同时 CYB-3798 的「下发历史」抽屉只列批次，单 run 不会出现——需要合并展示。

## Solution

**消息格式**：`{"asset_ids": ["a","b",...], "topic": "..."}`。`asset_ids` 驱动下发；`topic` 为**预留字段**（当前不使用，供未来路由/标注）；消息对未知字段前向宽松（忽略多余键），便于后续扩展。**不兼容**旧的单条 `{"asset_id":"x"}` 格式（按用户确认，无需兼容）。

**按消息判定**：一条消息 1 个资产 → 单 run；>1 → 批次。每个流水线绑定各下发一次（fan-out 不变）。

- 单 run 走 `CreateRunByTemplateID`，打 `Owner="subscription-task:<id>"`（`pipeline_runs.owner` 已有列）+ `TargetID`。
- 批次走既有 `CreateBatch`（`created_by="subscription-task:<id>"`）。
- 两者都可用 `createdBy` 反查 → 「下发历史」抽屉合并列出 run + 批次。

## Key Decisions

1. **按消息分组**（不是按整次 pull 合并）：`Pull` 返回 `[][]string`（每条消息一组资产），`executeTask` 逐消息决定 run/批次。发布方用「一次 publish」表达一个下发单元。
2. **不兼容旧格式 + 预留扩展**：只认 `asset_ids`；消息结构预留 `topic` 字段、对未知字段宽松（前向可扩展），不为旧 `asset_id` 单条格式做兼容（用户确认不需要）。
3. **复用 owner 反查**：单 run 用 `pipeline_runs.owner`、批次用 `backfill_jobs.created_by`，同一约定串 `subscription-task:<id>`；无新表、无 migration。
4. **Ack 语义不变**：保留 pull 级 Ack/Nack（任一消息下发失败 → nack 整个 pull 重投，at-least-once，与现状一致）。

## Scope

- **后端**：`subtask/pubsub.go`（解析数组 + 按消息分组）、`subtask/usecase.go`（`RunCreator` + 逐消息 run/批次）、`subtask/wiring.go` + `cmd/server/core.go`（接 `CreateRunByTemplateID`）、`pipeline` `ListRuns`/`PipelineRunListFilter` 加 `createdBy` 过滤。
- **契约**：openapi（`/runs` 加 `createdBy`）、api-guide、smoke、`subscription-task-integration.md`（消息格式）。
- **前端**：「下发历史」抽屉合并 run + 批次。
- **不做**：migration、鉴权改动、per-message 级 ack 拆分。
