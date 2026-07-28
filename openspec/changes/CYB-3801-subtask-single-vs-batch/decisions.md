# CYB-3801 Decisions

## 2026-07-22 — 按消息分组，而非整次 pull 合并

- **Context**: 现状把一次 pull 的所有消息资产合并成一个批次。新需求要「一条消息一个下发单元」。
- **Decision**: `Pull` 返回 `[][]string`（按消息分组），`executeTask` 逐消息判定 run/批次。
- **Alternatives**: 按整次 pull 的总资产数判定 —— 会把多条独立单资产消息误并成一个批次，违背发布方语义。
- **Rationale**: 发布方用「一次 publish」表达意图（单资产 or 一批），按消息才忠实。

## 2026-07-22 — 单 run 复用 pipeline_runs.owner 反查，不建历史表

- **Context**: 「下发历史」要同时列 run 和批次。
- **Decision**: 单 run 打 `owner="subscription-task:<id>"`，批次沿用 `created_by="subscription-task:<id>"`；前端分别用 `runs?createdBy=` 和 `backfill?createdBy=` 拉，再合并。
- **Alternatives**: 建订阅任务侧的 dispatch 历史表 —— 新表 + migration，过重。
- **Rationale**: 两处都已有可查询列，零 migration，最小改动。

## 2026-07-22 — 不兼容旧单条格式，消息预留扩展字段

- **Context**: 是否兼容旧 `{"asset_id":"x"}`；是否为将来留扩展位。
- **Decision**: 只认 `{"asset_ids":[...]}`，不为旧单条格式做兼容（用户确认不需要）。消息结构加**预留** `topic` 字段（当前仅解析+日志，不参与下发），并对未知键宽松（前向可扩展）。
- **Alternatives**: 同时接受 `asset_id` —— 用户明确不需要，徒增分支。
- **Rationale**: 更干净；`topic` 等预留字段为未来路由/标注留口，不引入当前复杂度。

## 2026-07-22 — Ack 保持 pull 级

- **Context**: 逐消息下发后，失败该怎样 ack。
- **Decision**: 保留 pull 级 Ack/Nack（任一失败 nack 整个 pull）。
- **Rationale**: 与 CYB-3778 现状一致；per-message ack 拆分复杂且收益低。at-least-once 重复下发的幂等性由批量/资产层承担（同现状）。
