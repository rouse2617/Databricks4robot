# Tasks — 2.0 Roadmap

> Source of truth: #[[file:docs/review/next-steps-tasks.md]]
> 设计文档: #[[file:docs/review/data-platform-design.md]], #[[file:docs/review/outbox-worker-design.md]]
>
> **同步规则**：每完成一个 task（`[ ]` → `[x]`），必须同步更新 #[[file:docs/review/next-steps-tasks.md]] 中对应条目的状态为 `done` 并补落地证据（PR 链接或 commit hash）。两边保持一致。

---

## Milestone 1: P0-3 — lifecycle_state 主消费切换

> 设计依据: data-platform-design.md §5.8.1

### 后端
- [ ] 1.1 `api/openapi.yaml`：`status`/`type`/`duration_sec` 加 `deprecated: true`
- [ ] 1.2 `internal/searchindex/builder.go`：ES aggregation 从 `status` 切到 `lifecycle_state`
- [ ] 1.3 `internal/handlers/search/handler.go`：`status_agg` → `lifecycle_state_agg`
- [ ] 1.4 `docs/review/api-guide.md`：更新示例响应，标注 deprecated 字段

### 前端 P0-FE-1: duration_sec → duration_ms
- [ ] 1.5 `AssetsResultsPane` 排序/时长列改读 `duration_ms`，UI 按秒展示（÷1000）
- [ ] 1.6 `OverviewTab` / `AssetPreviewHero` 改读 `duration_ms`
- [ ] 1.7 全仓 `rg "duration_sec"` 仅剩 SDK/API 兼容层

### 前端 P0-FE-2: lifecycle_state 主路径
- [ ] 1.8 列表过滤默认走 `lifecycle_state`
- [ ] 1.9 详情 badge 用 `lifecycle_state`，`status` 灰显 `(legacy)`
- [ ] 1.10 facet sidebar 切到 `lifecycle_state`

### 前端 P0-FE-3: 暴露新字段到 UI
- [ ] 1.11 `asset_type` 进列表可选列 + 详情概览 + 过滤入口
- [ ] 1.12 `retention_tier` 进列表可选列 + 详情概览 + 过滤入口
- [ ] 1.13 `expire_at` 进列表可选列 + 详情概览（相对时间）+ 过滤入口
- [ ] 1.14 `owner` 进列表可选列 + 过滤入口
- [ ] 1.15 vitest 通过

---

## Milestone 2: P0-4 — Outbox Worker 进程内 MVP

> 设计依据: outbox-worker-design.md §2–§11

### Worker 主循环
- [ ] 2.1 Drain loop：同一 tick 内反复 `processBatch` 直到 pending 空（§5.1）
- [ ] 2.2 Panic recovery：顶层 `defer recover` → `health.MarkUnhealthy` + 可选 `os.Exit(1)`（§5.1）
- [ ] 2.3 指数退避：PG fetch 失败时 1s→2s→5s→10s 封顶（§9）
- [ ] 2.4 部分失败：ES bulk 部分失败时按 doc 拆分 published/deferred（§5.1 步骤 5）

### Cursor 推进
- [ ] 2.5 `AssetEventRepo.ComputeSafeHorizon(ctx) (int64, error)`（§4.2）
- [ ] 2.6 `AssetEventRepo.MarkPublishedAndAdvanceCursor(ctx, eventSeqs, sinkName) error`（§4.2）
- [ ] 2.7 Worker 改用 `MarkPublishedAndAdvanceCursor`
- [ ] 2.8 `outbox_sink_cursors` seed：`INSERT ('es_assets', 0) ON CONFLICT DO NOTHING`

### 健康检查 + 配置
- [ ] 2.9 新增 `internal/outbox/health.go`
- [ ] 2.10 `routes/routes.go` 注册 `/healthz/outbox`
- [ ] 2.11 配置项：`OUTBOX_WORKER_ENABLED`/`BATCH_SIZE`/`TICK_INTERVAL`/`RETRY_LIMIT`/`FATAL_ON_PANIC`（§10）

### Metrics（轻量版 P1-2）
- [ ] 2.12 `outbox_worker_pending_total` gauge
- [ ] 2.13 `outbox_worker_batch_duration_ms` histogram
- [ ] 2.14 `outbox_oldest_pending_age_seconds` gauge
- [ ] 2.15 `outbox_es_bulk_failures_total` / `outbox_bulk_partial_failure_total` counters

### 验证
- [ ] 2.16 `go build` + `go vet` + `go test` 全过
- [ ] 2.17 `OUTBOX_WORKER_ENABLED=false` 行为不变
- [ ] 2.18 本地 docker compose：写 event → ≤60s ES 可查

---

## Milestone 3: P0-6 — PgBouncer 入栈

> 设计依据: data-platform-design.md §6.3

- [ ] 3.1 `deploy/local/pgbouncer/pgbouncer.ini`（transaction pool）
- [ ] 3.2 `deploy/local/pgbouncer/userlist.txt`
- [ ] 3.3 `docker-compose.yml` + `docker-compose.all.yml` 加 pgbouncer 服务
- [ ] 3.4 `backend/.env.example` 更新 `DB_HOST=pgbouncer`、`DB_PORT=6432`
- [ ] 3.5 验证：`make dev-up` 正常连接

---

## Milestone 4: P0-T — 测试同窗口

> 设计依据: outbox-worker-design.md §12

### P0-T-1: Outbox 集成测试（blocked-by M2）
- [ ] 4.1 `internal/outbox/e2e_test.go`（`//go:build integration`）
- [ ] 4.2 testcontainers-go 起 PG + ES
- [ ] 4.3 `TestE2E_Notify_HappyPath`
- [ ] 4.4 `TestE2E_DedupBatch`
- [ ] 4.5 `TestE2E_RestartReplay`
- [ ] 4.6 `TestE2E_ConcurrentAck_NoSeqGap`
- [ ] 4.7 `TestE2E_ESDown_Backpressure`

### P0-T-2: CI 集成（blocked-by 4.1–4.7）
- [ ] 4.8 GitHub Actions `test-integration` job
- [ ] 4.9 testcontainers 镜像 cache

### P0-T-3: Backfill 对账
- [ ] 4.10 `scripts/backfill-audit.sh`
- [ ] 4.11 新字段空值率 < 0.1%，双写一致率 100%

### P0-T-4: 前端覆盖率守门
- [ ] 4.12 vitest coverage 配置
- [ ] 4.13 核心组件 ≥ 70%，CI gate

---

## Milestone 5: P1-1/P1-2 — ES 闸口 + 监控

> 设计依据: outbox-worker-design.md §6/§7/§11, data-platform-design.md §8

### P1-1: ES 索引闸口
- [ ] 5.1 确认 ES mapping 与 outbox-worker-design.md §6 一致
- [ ] 5.2 `scripts/es-pg-audit.sh`：PG↔ES 一致率 > 99.9%
- [ ] 5.3 CI nightly 对账 job

### P1-2: 监控指标
- [ ] 5.4 `internal/metrics/outbox.go` 按 §11.1 定义全部 Prometheus metrics
- [ ] 5.5 `backend/README.md` 告警阈值建议表

---

## Milestone 6: P1-3~P1-6 — Iceberg 湖仓

> 设计依据: data-platform-design.md §5.6, §5.11, §6.2

### P1-3: Catalog 部署
- [ ] 6.1 选型 ADR（Polaris vs Lakekeeper）
- [ ] 6.2 docker-compose 加 Catalog 服务
- [ ] 6.3 PyIceberg + Trino 冒烟

### P1-4: Bronze 入湖
- [ ] 6.4 Outbox Worker Bronze Sink：写 staging parquet
- [ ] 6.5 PyIceberg CronJob：MERGE INTO bronze（按 event_seq 去重）
- [ ] 6.6 24h 无丢失验证

### P1-5: Compact
- [ ] 6.7 Expire Snapshots + Rewrite Data Files + Remove Orphan Files
- [ ] 6.8 7 天后小文件数稳定

### P1-6: Trino 接入
- [ ] 6.9 `internal/trino/client.go` 对接真实 catalog
- [ ] 6.10 `/api/v1/lakehouse/sync-status|training-assets|quality-distribution` 返回真实数据

---

## Milestone 7: P1-7/P1-8 — 工程基建

> 设计依据: data-platform-design.md §5.9, outbox-worker-design.md §17

### P1-7: 事件 Schema CI 守门
- [ ] 7.1 `schemas/events/` 目录 + JSON Schema 文件
- [ ] 7.2 CI workflow
- [ ] 7.3 minor + major bump 演练
- [ ] 7.4 `CLAUDE.md` 更新

### P1-8: Worker 独立进程（blocked-by P0-4 稳定 90 天）
- [ ] 7.5 `cmd/outbox-worker/main.go`
- [ ] 7.6 独立 Dockerfile + K8s Deployment
- [ ] 7.7 切换验证：无丢事件

---

## Milestone 8: P1-FE — 前端 2.0

> 设计依据: data-platform-design.md §5.3, api-guide.md

### P1-FE-1: Lakehouse Dashboard（blocked-by M6）
- [ ] 8.1 `AnalyticsPage` 接入 lakehouse 端点，≥ 2 个可视化卡片

### P1-FE-2: 同步状态可视化
- [ ] 8.2 接入 `/api/v1/lakehouse/sync-status`，红绿灯状态卡片

### P1-FE-3: Admin Reindex 入口
- [ ] 8.3 SettingsPage "重建 ES 索引"按钮（dry_run + 二次确认）

### P1-FE-4: 保留期/过期视图
- [ ] 8.4 "30 天内将过期"快捷 chip + retention badge

---

## Milestone 9: P1-T — 测试 2.0

> 设计依据: outbox-worker-design.md §12, data-platform-design.md §8.2

- [ ] 9.1 P1-T-1: PG↔ES 对账脚本 + CI nightly
- [ ] 9.2 P1-T-2: Outbox 性能压测（50 events/s × 60s → P99 ≤ 60s）
- [ ] 9.3 P1-T-3: SDK 集成测试（pytest E2E）
- [ ] 9.4 P1-T-4: 事件 schema CI 守门测试（blocked-by 7.1–7.2）
- [ ] 9.5 P1-T-5: 前端 E2E（Playwright 1 条 happy-path）

---

## Milestone 10: P2 — 打磨（按触发条件启动）

> 设计依据: next-steps-tasks.md §3

### 后端
- [ ] 10.1 P2-1: `POST /api/v1/assets:batch_get` 批量查询
- [ ] 10.2 P2-2: ES nested query 同 tag 内多条件
- [ ] 10.3 P2-3: ES `tagged_at` 字段
- [ ] 10.4 P2-4: `lifecycle_state` CHECK 约束
- [ ] 10.5 P2-5: 资产详情 fan-out 单 SQL 化
- [ ] 10.6 P2-6: DLQ 表
- [ ] 10.7 P2-7: 事件 retention 策略
- [ ] 10.8 P2-9: 彻底移除 `cf_*` 遗留层

### 前端
- [ ] 10.9 P2-FE-1: Tag 高级筛选 UI
- [ ] 10.10 P2-FE-2: 通用事件流总览页

### 测试
- [ ] 10.11 P2-T-1: Frontend coverage ≥ 85%
- [ ] 10.12 P2-T-2: Outbox chaos 测试
