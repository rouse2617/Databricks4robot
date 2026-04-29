# Requirements — P0-4: Outbox Worker 进程内 MVP

## Introduction

补齐 Outbox Worker 实现，使其满足 `outbox-worker-design.md` G1–G5 验收标准。当前 `internal/outbox/worker.go` 已有基础骨架，但缺少 drain loop、panic recovery、safe_horizon cursor 推进、健康检查端点等设计文档要求的能力。

参考文档：
- `docs/review/outbox-worker-design.md`（完整设计）
- `docs/review/data-platform-design.md` §5.6.2
- `docs/review/next-steps-tasks.md` P0-4

## Requirements

### R1: Drain Loop
每个 tick 内反复 fetch 直到 pending 空，不靠下一次 tick 推进。burst 写入不被 30s 节奏限流。

### R2: Panic Recovery
顶层 `recover` → 健康降级 + 可选 `os.Exit(1)` 让 K8s liveness 重启容器。

### R3: Safe Horizon Cursor 推进
实现 `outbox_sink_cursors` 表的 `safe_horizon = MIN(pending) - 1` 协议，cursor 单调守卫。

### R4: 健康检查端点
`/healthz/outbox` 端点，worker panic 或长时间无消费时报 unhealthy。

### R5: 指数退避
PG fetch 失败时指数退避 1s → 2s → 5s → 10s（封顶），不退出循环。

### R6: 部分失败处理
ES bulk 部分失败时，失败 doc 对应 events 保留 pending，成功 doc 对应 events 标 published。

### R7: G1–G5 验收
- G1: 端到端 P99 < 60s（tick + drain）
- G2: Worker 重启不丢事件
- G3: 同一 asset 多次更新合并为最少 ES 写次数
- G4: ES 索引误删后可从任意 event_seq 重放重建
- G5: Cursor 推进不丢事件（乱序 ack 安全）
