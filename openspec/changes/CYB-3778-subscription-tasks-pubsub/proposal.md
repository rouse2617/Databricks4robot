# CYB-3778: Subscription Tasks — Pub/Sub Architecture

## Problem

CYB-3744 的定时任务用 REST 轮询拉取 asset ID，耦合上游 API 格式。用户希望改成 pub/sub 推送模式：外部系统主动推 asset ID，databrew 被动消费。

## Solution

用 GCP Pub/Sub 替代 REST source。用户在 GCP 建 topic/subscription，在 databrew 填 subscription ID，外部系统往 topic 发消息，databrew 自动 pull → 建批次。

## Key Decisions

1. **用户自建 topic** — 不由 databrew 托管（省 SA 权限，灵活）
2. **统一消息格式** — `{"asset_id": "xxx"}`，不搞 id_path 灵活配置
3. **Pull 模式** — 复用 ticker 架构（不引入 streaming goroutine），默认 10s 间隔
4. **删除全部 REST 代码** — schedtask 包整体替换为 subtask 包
5. **Ack 语义** — 成功建批次后 ack，失败 nack（消息自动重试）

## Scope

- Drop `scheduled_tasks` 表 → `subscription_tasks`
- Delete `schedtask/` → new `subtask/` package
- API: `/subscription-tasks` (CRUD + pause/resume)
- Frontend: 订阅任务 tab
- Integration doc for publishers
