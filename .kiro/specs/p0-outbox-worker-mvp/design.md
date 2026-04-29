# Design — P0-4: Outbox Worker 进程内 MVP

## Overview

补齐 `internal/outbox/worker.go` 使其满足 #[[file:docs/review/outbox-worker-design.md]] 的完整规格。当前实现是简化版骨架，本 spec 补齐 drain loop、panic recovery、safe_horizon cursor 推进、健康检查、指数退避、部分失败处理。

## Source of Truth

- 架构决策：#[[file:docs/review/data-platform-design.md]] §5.6.2（Transactional Outbox）
- Worker 工程设计：#[[file:docs/review/outbox-worker-design.md]]（完整伪代码、cursor 协议、错误策略、metrics、验收标准）
- Schema：#[[file:schemas/pg-phase0.sql]] `asset_events` + `outbox_sink_cursors` 表

## 变更范围

### `internal/outbox/worker.go`
1. **Drain loop**：同一 tick 内反复 `processBatch` 直到 pending 空（§5.1 伪代码）
2. **Panic recovery**：顶层 `defer recover` → `health.MarkUnhealthy` + 可选 `os.Exit(1)`（§5.1）
3. **Safe horizon cursor**：`MarkPublishedAndAdvanceCursor` 在单事务内更新 `asset_events` 与 `outbox_sink_cursors`（§4.2）
4. **指数退避**：PG fetch 失败时 1s → 2s → 5s → 10s 封顶（§9 错误策略）
5. **部分失败**：ES bulk 部分失败时按 doc 拆分 published/deferred（§5.1 步骤 5）

### `internal/outbox/health.go`（新增）
- 健康状态管理，供 `/healthz/outbox` 端点查询

### `routes/routes.go`
- 注册 `/healthz/outbox` 端点

### `internal/postgres/repos.go`
- `AssetEventRepo` 新增 `ComputeSafeHorizon` 方法（§4.2 SQL）
- `AssetEventRepo` 新增 `MarkPublishedAndAdvanceCursor` 方法

### 配置
- 环境变量：`OUTBOX_WORKER_ENABLED`、`OUTBOX_BATCH_SIZE`、`OUTBOX_TICK_INTERVAL`、`OUTBOX_RETRY_LIMIT`、`OUTBOX_FATAL_ON_PANIC`（§10）

## 验收标准

严格按 #[[file:docs/review/outbox-worker-design.md]] §1.1 G1–G5：
- G1: 端到端 P99 < 60s
- G2: Worker 重启不丢事件
- G3: 同 asset 多次更新合并为最少 ES 写次数
- G4: ES 索引误删后可从任意 event_seq 重放重建
- G5: Cursor 推进不丢事件（乱序 ack 安全）
