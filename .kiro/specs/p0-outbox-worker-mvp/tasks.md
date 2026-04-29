# Tasks — P0-4: Outbox Worker 进程内 MVP

> Source of truth: #[[file:docs/review/outbox-worker-design.md]]

## Milestone 1: Worker 主循环补齐

- [ ] 1.1 实现 drain loop：同一 tick 内反复 `processBatch` 直到 pending 空或出错（§5.1）
- [ ] 1.2 实现 panic recovery：顶层 `defer recover` → `health.MarkUnhealthy` + 可选 `os.Exit(1)`（§5.1）
- [ ] 1.3 实现指数退避：PG fetch 失败时 1s → 2s → 5s → 10s 封顶，不退出循环（§9）
- [ ] 1.4 实现部分失败处理：ES bulk 部分失败时，失败 doc 对应 events 保留 pending，成功 doc 标 published（§5.1 步骤 5）

## Milestone 2: Cursor 推进协议

- [ ] 2.1 `AssetEventRepo` 新增 `ComputeSafeHorizon(ctx) (int64, error)` 方法（§4.2 SQL）
- [ ] 2.2 `AssetEventRepo` 新增 `MarkPublishedAndAdvanceCursor(ctx, eventSeqs, sinkName) error` 方法（§4.2）
- [ ] 2.3 Worker `processBatch` 改用 `MarkPublishedAndAdvanceCursor` 替代当前的 `MarkPublished`（§5.1 步骤 6）
- [ ] 2.4 `outbox_sink_cursors` 表初始化 seed：`INSERT INTO outbox_sink_cursors (sink_name, last_published_seq) VALUES ('es_assets', 0) ON CONFLICT DO NOTHING`

## Milestone 3: 健康检查 + 配置

- [ ] 3.1 新增 `internal/outbox/health.go`：健康状态管理（healthy/unhealthy/degraded）
- [ ] 3.2 `routes/routes.go` 注册 `/healthz/outbox` 端点
- [ ] 3.3 配置项：`OUTBOX_WORKER_ENABLED`、`OUTBOX_BATCH_SIZE`（默认 1000）、`OUTBOX_TICK_INTERVAL`（默认 30s）、`OUTBOX_RETRY_LIMIT`（默认 10）、`OUTBOX_FATAL_ON_PANIC`（默认 true）（§10）
- [ ] 3.4 `internal/config/config.go` 新增 outbox 相关配置字段

## Milestone 4: Metrics（轻量版 P1-2）

- [ ] 4.1 `outbox_worker_pending_total` gauge（§11.1）
- [ ] 4.2 `outbox_worker_batch_duration_ms` histogram（§11.1）
- [ ] 4.3 `outbox_oldest_pending_age_seconds` gauge（§11.1）
- [ ] 4.4 `outbox_es_bulk_failures_total` counter（§11.1）
- [ ] 4.5 `outbox_bulk_partial_failure_total` counter（§11.1）

## Milestone 5: 验证

- [ ] 5.1 `go build ./...` + `go vet ./...` + `go test ./...` 全过
- [ ] 5.2 `OUTBOX_WORKER_ENABLED=false` 时 backend 行为与现状完全一致
- [ ] 5.3 本地 docker compose 全链路验证：写 event → ≤60s ES 可查
