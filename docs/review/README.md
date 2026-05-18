# 评审文档包 · 数据平台 (cyber-databrew)

本目录是面向方案评审与团队交付的完整文档集合，**自包含、可独立阅读**。

## 当前架构基线

> **当前位置：2.0**（PostgreSQL + Elasticsearch + BigQuery + Outbox CDC）。
>
> - 主库：PostgreSQL
> - 搜索：Elasticsearch（Outbox relay/subscriber 驱动同步）
> - 湖仓：BigQuery + BigLake-managed Iceberg（Cloud Run Job 入湖）
> - `asset_events` 是事件主线起点；生产主线为 **Outbox relay/subscriber**（Pub/Sub → ES）

## 阅读顺序

| 序号 | 文档 | 内容 | 推荐读者 |
|------|------|------|----------|
| 1 | `data-platform-design.md` | 数据平台整体方案设计：架构 / 核心表 / 同步 / API / 部署 / 可靠性 | **必读**，所有评审人 |
| 2 | `schema-reference.md` | PG 全表字段速查 + 上线优先级 | DBA / 后端 / 数据工程 |
| 3 | `api-guide.md` | 全部 REST 端点 curl 示例、错误码、工作流 | SDK / 前端 / 集成 |
| 4 | `algo-lifecycle-and-data-model.md` | 算法生命周期、状态机、依赖解锁、数据模型 | 算法 / 后端 |
| 5 | `use-cases.md` | 全部角色 × 业务域 use case 矩阵 | 前端 / SDK / 后端 |
| 6 | `eval-metrics-design.md` | Eval/Metrics 专项：表结构、事件、API、Phase 1.5 | 后端 / 数据工程 / 算法 |
| 7 | `grace-migration-notes.md` | cyber-grace ↔ 本平台字段与幂等对照 | 集成 / 后端 |
| 8 | `sql.md` | Schema companion：DDL/字段 section 编号 | 后端 / DBA |
| 9 | `mcap-index-metadata-v1.md` | MCAP 索引元数据入库规范（V1） | 后端 / 数据工程 |
| 10 | `outbox-test-plan.md` | Outbox relay/subscriber 测试计划 | 后端 |
| 11 | `outbox-watermark-alerts.md` | Outbox watermark 告警阈值 | 后端 / SRE |
| 12 | `lakehouse-incremental-ingestion.md` | PG → Iceberg Bronze 增量入湖（已部署） | 数据工程 |
| 13 | `roadmap.md` | 渐进式开发路线图（v2） | 所有人 |

## 这份文档包覆盖了什么

| 评审议题 | 看哪一篇 |
|----------|----------|
| 整体架构 / 分层 / 组件职责 | 1. data-platform-design §4 |
| 资产概念定义 / 状态机 | 1. data-platform-design §5.1 + 4. algo-lifecycle |
| 表 schema / 字段定义 / 索引 | 1. data-platform-design §5.2 + 2. schema-reference |
| 数据同步机制（Outbox 主线） | 1. data-platform-design §5.6.2 + 10. outbox-test-plan |
| 事件契约 / payload schema 演进 | 1. data-platform-design §5.9 |
| API 设计 / 错误码 / 幂等 | 1. data-platform-design §5.8 + 3. api-guide |
| Grace 生产库 ↔ 本平台字段 | 7. grace-migration-notes |
| 算法接入 / 状态流转 / 依赖编排 | 4. algo-lifecycle |
| 湖仓入湖 / 查询 | 12. lakehouse-incremental-ingestion + 1. data-platform-design §5.6 |
| 部署形态 | 1. data-platform-design §6 |
| 上线计划 / Phase 拆分 | 13. roadmap |

## 文档维护说明

| 文档 | 类型 | 维护节奏 | 权威性 |
|------|------|----------|--------|
| `data-platform-design.md` | 阶段性设计快照 | 架构重大变化时更新 | 评审基线，架构主口径 |
| `schema-reference.md` | 字段速查 | 表/字段变化时更新 | DDL 以 `schemas/pg-phase0.sql` 为源 |
| `api-guide.md` | API 使用指南 | 长期维护 | API 以 `api/openapi.yaml` 为源 |
| `algo-lifecycle-and-data-model.md` | 算法子领域设计 | 长期维护 | 与主设计冲突时以主设计为准 |
| `sql.md` | DDL 伴随文档 | 仅表结构变化时更新 | 不再承载同步链路设计 |
| `roadmap.md` | 路线图 | Phase 完成后更新 | 当前交付优先级参考 |

历史设计文档已归档至 `docs/archive/review/`；仓库内 `docs/archive/` 其他文档为历史调研。如与本目录冲突，**以本目录为评审基线**。
