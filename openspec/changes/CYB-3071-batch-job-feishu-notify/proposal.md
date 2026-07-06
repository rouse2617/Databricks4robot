# Proposal — CYB-3071

## Why

批量任务（backfill job）跑完之后，用户只能靠反复刷页面才知道结果，没有主动通知。

## What Changes

### New Capabilities
- 批量任务状态从非终态跨越到终态（`completed`/`failed`）时，系统发送一条飞书文本消息，内容包含总数/成功数/失败数与任务详情链接——不论结果好坏都发送一次，不只在失败时发。

### Modified Capabilities
无——这是纯新增能力，不修改现有批量任务状态机行为。

## Impact
- **Affected code**: `backend/internal/usecase/backfill/`, `backend/internal/notify/feishu/`（新包）, `backend/internal/config/`
- **New APIs**: 无（纯后端内部行为，不新增/修改任何 HTTP 接口）
- **Dependencies**: 无新增第三方依赖——手写飞书 webhook 客户端，复用 `.github/scripts/feishu-notify.sh` 已在生产验证过的 payload 格式
- **Schema**: `backfill_jobs` 新增可空列 `notification_sent_at TIMESTAMPTZ`，需要走迁移审批

## Scope

- **In scope**:
  - 新增 `backend/internal/notify/feishu` 包：`Config{WebhookURL, Timeout, MaxRetries}`、`Sender` 接口、`Client.SendText(ctx, text) error`。
  - `backfill_jobs` 加 `notification_sent_at` 列，作为跨实例安全的"是否已通知"原子认领标记。
  - 在批量任务状态计算处（跨越终态边界时）触发一次性、同步的飞书通知。
  - 新配置项 `BACKFILL_NOTIFY_FEISHU_WEBHOOK_URL`（空值=功能关闭）与通用的 `FRONTEND_BASE_URL`（用于拼任务详情链接）。

- **Out of scope**（讨论中明确排除，非遗漏）：
  - 流水线 run/step 级失败通知——单独作为未来的需求，不在这次范围内。
  - 通知发送失败后的跨 tick 重试机制——批量任务终态之后不会再被 watcher 重新扫描，没有"下一轮 tick"，这次接受一次性最佳努力发送；如果后续证明是真问题再补重试扫描。
  - Argo Events / GCP Cloud Monitoring 等平台原生告警方案——已比较过，覆盖不了批量任务这个纯 Postgres 状态事件，且引入新基础设施成本不划算。
  - 引入 `go-lark` 等第三方飞书 SDK——需求足够窄（纯文本、单一 webhook），手写更省心。

## Success Criteria
- [ ] 批量任务从 `running` 变成 `completed` 或 `failed` 时，配置的飞书群收到一条包含总数/成功/失败计数和任务链接的消息。
- [ ] 同一个批量任务的终态跨越只触发一次通知，即使多个后端实例同时观察到这次转变。
- [ ] `BACKFILL_NOTIFY_FEISHU_WEBHOOK_URL` 为空时，功能完全不触发任何网络请求（no-op）。
- [ ] `notify/feishu` 包与 `backfill` 用例包之间无领域耦合——`feishu` 包不引用任何 batch/pipeline 相关类型。
