# Requirements — 2.0 Roadmap（全量未完成任务）

## Introduction

本 spec 合并 #[[file:docs/review/next-steps-tasks.md]] 中所有未完成任务（P0 收尾 → P1 关键路径 → P2 打磨），作为 1.0 → 2.0 → 2.0 打磨的统一执行清单。

Source of truth（按章节）：
- 整体架构与演进：#[[file:docs/review/data-platform-design.md]]
- Outbox Worker 工程：#[[file:docs/review/outbox-worker-design.md]]
- Schema 字段定义：#[[file:docs/review/schema-reference.md]]
- API 使用指南：#[[file:docs/review/api-guide.md]]
- 算法生命周期：#[[file:docs/review/algo-lifecycle-and-data-model.md]]
- 前端设计规范：#[[file:.kiro/steering/frontend-design-rules.md]]

## 闸口（Phase Gate）

| 闸口 | 触发条件 | 验收文档 |
|------|----------|----------|
| 1.0 → 2.0 | P0 全绿 + P1-1/P1-2 通过 | data-platform-design.md §9.1.1 |
| 2.0 → 3.x | Iceberg 入湖 90 天稳定 + 第一个 dataset 上线 | data-platform-design.md §9.1.2 |

---

## P0 — 1.0 收尾 / 2.0 启用前置

### P0-3: lifecycle_state 主消费切换
- **设计依据**：data-platform-design.md §5.8.1（字段退役流程）
- API `status`/`type`/`duration_sec` 加 OpenAPI `deprecated: true`
- ES facet 从 `status` 切到 `lifecycle_state`
- 前端列表/详情/facets 默认用 `lifecycle_state`

### P0-4: Outbox Worker 进程内 MVP
- **设计依据**：outbox-worker-design.md（全文）
- 补齐 drain loop、panic recovery、safe_horizon cursor 推进、健康检查、指数退避、部分失败处理
- 满足 G1–G5 验收标准

### P0-6: PgBouncer 入栈
- **设计依据**：data-platform-design.md §6.3
- Docker Compose + K8s 加 PgBouncer（transaction pool）
- Backend 改指向 PgBouncer，业务代码 0 改动

### P0-FE-1: duration_sec → duration_ms 全面切换
- 全仓 `rg "duration_sec"` 仅剩 SDK/API 兼容层

### P0-FE-2: lifecycle_state 前端主路径
- 列表过滤、详情 badge、facet 从 `status` 切到 `lifecycle_state`

### P0-FE-3: 暴露新字段到 UI
- `asset_type / retention_tier / expire_at / owner` 进列表列 + 详情 + 过滤

### P0-T-1: Outbox Worker 集成测试
- **设计依据**：outbox-worker-design.md §12
- testcontainers-go 五件套（HappyPath / DedupBatch / RestartReplay / ConcurrentAck / ESDown）

### P0-T-2: CI 接集成测试
- GitHub Actions 矩阵跑 unit + integration

### P0-T-3: Backfill 对账脚本
- `scripts/backfill-audit.sh`，新字段空值率 < 0.1%

### P0-T-4: 前端单测覆盖率守门
- vitest coverage gate，核心组件 ≥ 70%

---

## P1 — 2.0 启用关键路径

### P1-1: ES 索引正式启用
- **设计依据**：outbox-worker-design.md §6/§7
- PG↔ES 一致率 > 99.9%

### P1-2: Outbox 监控 + 告警
- **设计依据**：outbox-worker-design.md §11，data-platform-design.md §8
- Prometheus metrics + 告警阈值文档

### P1-3: Iceberg REST Catalog 选型 + 部署
- **设计依据**：data-platform-design.md §5.6.4，§5.11
- Polaris（首选）或 Lakekeeper

### P1-4: PyIceberg Bronze 入湖 CronJob
- **设计依据**：data-platform-design.md §5.6.2（两段式入湖）

### P1-5: PyIceberg Compact CronJob
- 合并小文件、过期 snapshot 清理

### P1-6: Trino 查询接入
- `/api/v1/lakehouse/*` 对接真实 catalog

### P1-7: 事件 Schema CI 守门
- **设计依据**：data-platform-design.md §5.9

### P1-8: Worker 独立进程
- **设计依据**：outbox-worker-design.md §17
- blocked-by P0-4 稳定 90 天

### P1-FE-1~4: 前端（Lakehouse Dashboard / 同步状态 / Admin Reindex / 保留期视图）
### P1-T-1~5: 测试（PG↔ES 对账 / 性能压测 / SDK E2E / Schema CI 测试 / 前端 E2E）

---

## P2 — 打磨

- P2-1: 批量查询 API
- P2-2: ES nested query 同 tag 内多条件
- P2-3: ES tagged_at 字段
- P2-4: lifecycle_state CHECK 约束
- P2-5: 资产详情 fan-out 单 SQL 化
- P2-6: DLQ 表
- P2-7: 事件 retention 策略
- P2-9: 彻底移除 cf_* 遗留层
- P2-FE-1: Tag 高级筛选 UI
- P2-FE-2: 通用事件流总览页
- P2-T-1: Frontend coverage ≥ 85%
- P2-T-2: Outbox chaos 测试
