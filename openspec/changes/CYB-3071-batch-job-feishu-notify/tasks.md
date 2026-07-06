# Tasks — CYB-3071

## Implementation
- [ ] [backend] 新增迁移：`backfill_jobs` 加可空列 `notification_sent_at TIMESTAMPTZ`。
- [ ] [backend] 新增 `backend/internal/notify/feishu` 包：`Config`、`Sender` 接口、`Client.SendText`（`WebhookURL` 空值 no-op；`Timeout`/`MaxRetries` 有默认值；同一调用内重试网络抖动）。
- [ ] [backend] `backend/internal/config/config.go` 新增 `BackfillNotifyFeishuWebhookURL`、`FrontendBaseURL` 配置字段。
- [ ] [backend] `backfill.Usecase` 新增 `notifier Sender` 字段与注入方式（参考 `SetArgoRunWebhook` 模式）。
- [ ] [backend] 在终态跨越检测处（`usecase.go` 约 1424 行附近）新增：原子认领 `UPDATE backfill_jobs SET notification_sent_at = now() WHERE id = $1 AND notification_sent_at IS NULL RETURNING id` → 认领成功则拼消息文本并调 `notifier.SendText`。
- [ ] [backend] `cmd/server` 组装代码：读取配置构造 `feishu.Client`，注入到 `backfill.Usecase`。

## Local verification
- [ ] `cd backend && go test ./internal/notify/... ./internal/usecase/backfill/...`
- [ ] `cd backend && go test ./...`
- [ ] `make fmt && make vet`

### 测试要点
- [ ] `feishu` 包：用 `httptest.Server` 验证 payload 形状、超时行为、`MaxRetries` 次数、`WebhookURL` 为空时不发任何请求。
- [ ] `backfill` 用例：并发调用同一个任务的终态转变检测，验证注入的假 `Sender` 只被调用一次（模拟多实例竞态）。
- [ ] `backfill` 用例：验证已经终态的任务再次被处理时（如果发生）不会重复触发通知（`notification_sent_at` 已非空即跳过）。
- [ ] `backfill` 用例：验证消息文本包含正确的总数/成功/失败计数与链接。

## API contract sync
不适用——本次改动不新增/修改任何 HTTP 接口。

## Deploy verification
- [ ] `bash scripts/apply-migration-dev.sh` 应用新迁移到 dev。
- [ ] 部署 backend dev。
- [ ] 配置 dev 环境的 `BACKFILL_NOTIFY_FEISHU_WEBHOOK_URL`（复用现有 CI 飞书群 webhook）与 `FRONTEND_BASE_URL`。
- [ ] 在 dev 上跑一个真实批量任务到终态（成功和失败各验证一次），确认飞书群收到消息，内容与链接正确。
- [ ] 确认 `backfill_jobs.notification_sent_at` 在通知后被正确置为非空。

## PR
- [ ] PR 描述包含 Linear ID（CYB-3071）与 OpenSpec change-id。
- [ ] 说明范围收窄的决定（流水线 run 级通知暂不做）与理由。
