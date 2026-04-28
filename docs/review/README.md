# 评审文档包 · 数据平台 (Databricks4robot)

本目录是面向方案评审与团队交付的完整文档集合，**自包含、可独立阅读**。

## 当前架构基线

> **当前位置：1.0**（仅 PostgreSQL 单库 + Backend，2.0 尚未启动）。
>
> 本文档包同时描述 **Current（1.0 已落地形态）** 与 **Target（2.0 / 3.x 演进目标）** 两套口径；遇到冲突时，所有四篇文档都以下面这条主线为准：
>
> - 主库：PostgreSQL（不再使用 Bigtable）
> - 当前态：标量列 + `asset_tags` / `asset_algo_latest` 投影表
> - 历史 / 同步：`asset_events` 统一事件表（outbox 起点）
> - JSONB 列（`cf_meta` / `cf_algo` / `cf_tag`）只在 1.0 → 2.0 迁移窗口内作为兼容路径保留
>
> 任何文档片段如果还在描述 "cf_algo 是主路径"、"Phase 1 切 Bigtable" 之类的形态，都属于**历史路径说明**，不是当前主架构。

## 阅读顺序

| 序号 | 文档 | 内容 | 推荐读者 |
|------|------|------|----------|
| 1 | [`data-platform-design.md`](./data-platform-design.md) | 数据平台整体方案设计：背景 / 架构 / 核心表 / 字段 / 数据同步 / API / 部署 / 可靠性 / 监控 / 实施计划 / 风险 | **必读**，所有评审人 |
| 2 | [`schema-reference.md`](./schema-reference.md) | PG 全表字段速查 + 上线优先级（Tier 1–5）+ 上线最小 checklist | DBA / 后端 / 数据工程 |
| 3 | [`api-guide.md`](./api-guide.md) | 全部 REST 端点的 curl 示例、错误码参考、典型工作流；含**新旧字段命名对照表**（lifecycle_state / asset_type / duration_ms） | 接 SDK / 前端 / 上下游集成 |
| 4 | [`algo-lifecycle-and-data-model.md`](./algo-lifecycle-and-data-model.md) | 算法生命周期、状态机、依赖解锁、数据模型细节（主线：`asset_algo_latest + asset_events`；附录 A 为 Phase 0 `cf_algo` 兼容路径） | 算法 / 后端 |

## 这份文档包覆盖了什么

| 评审议题 | 看哪一篇 |
|----------|----------|
| 整体架构 / 分层 / 组件职责 | 1. data-platform-design §4 |
| Bigtable / Postgres / Spanner 选型 | 1. data-platform-design §5.4 |
| 资产概念定义 / 状态机 | 1. data-platform-design §5.1 + 4. algo-lifecycle |
| 表 schema / 字段定义 / 索引 | 1. data-platform-design §5.2 + 2. schema-reference |
| 数据同步机制（不轮询，outbox + LISTEN/NOTIFY） | 1. data-platform-design §5.6.2 |
| 事件契约 / payload schema 演进 | 1. data-platform-design §5.9 |
| 一致性语义 / 重试 / DLQ / 对账 / 回放 | 1. data-platform-design §5.10 |
| 入湖 / ES 同步流水 | 1. data-platform-design §5.6.2 + §6 |
| API 设计 / 错误码 / 幂等 / 乐观锁 | 1. data-platform-design §5.8 + 3. api-guide |
| API 版本治理 / 字段退役流程 | 1. data-platform-design §5.8.1 + 3. api-guide 顶部对照表 |
| 算法接入 / 状态流转 / 依赖编排 | 4. algo-lifecycle |
| 部署形态 | 1. data-platform-design §6 |
| 故障域 / 冗余 / SLO | 1. data-platform-design §7、§8 |
| 安全 / 权限 / 密钥 / 数据分级 / 删除 SLA | 1. data-platform-design §7.1 |
| 上线计划 / Phase 拆分 | 1. data-platform-design §9 + 2. schema-reference 上线优先级 |
| Phase Gate（验收 + 回滚） | 1. data-platform-design §9.5 |
| 风险与未决事项（含 owner / 关闭标准） | 1. data-platform-design §10.1 |
| PII / GDPR 策略 | 1. data-platform-design §7.1.4 |

## 这份文档包不覆盖的（评审外议题）

- 容量预算 / 1x/3x/10x 资源模型 —— 待 2.0 真流量上线后由独立 Capacity ADR 承载
- RTO/RPO 量化 / 容灾演练计划 —— 由独立 SRE 文档承载（需先有客户 SLA 输入）
- oncall runbook / 告警处理流程 —— 由运维手册承载
- RACI / 团队职责边界 —— 团队稳定后单独 ownership map
- PG / ES / Iceberg 物理调优细节（索引 / 分区 / compact 节奏） —— 各自上线前发实施 ADR
- 多模态数据集物理格式（Daft / Lance）的具体落地 —— Phase 2+ 单独评审
- 训练平台 / 特征平台 / 实验分支管理 —— 独立后续设计
- PII / GDPR 删除链路细则 —— 单独治理文档（待写）
- 上云 Catalog 选型（Polaris / Gravitino / 云原生）的最终决策 —— 上云前 ADR

## 文档维护说明

四篇文档定位与生命周期：

| 文档 | 类型 | 维护节奏 | 权威性 |
|------|------|----------|--------|
| `data-platform-design.md` | 阶段性设计快照（1.0 → 3.x 整体演进） | 架构发生重大变化时整篇出新版本 | 评审基线，**架构主口径**以这篇为准 |
| `schema-reference.md` | 字段速查 + 上线优先级 | 表 / 字段变化时跟随更新 | DDL / 字段定义以 [`schemas/pg-phase0.sql`](../../schemas/pg-phase0.sql) 为最终源；本文档为人类可读视图 |
| `api-guide.md` | API 使用指南 + curl 示例 | 长期维护，随后端 API 演进同步更新 | API 行为最终以 [`api/openapi.yaml`](../../api/openapi.yaml) 为源 |
| `algo-lifecycle-and-data-model.md` | 算法子领域设计 | 长期维护，随算法生命周期 / 状态机演进同步更新 | 与 `data-platform-design.md` 冲突时以**整体设计文档为准** |

仓库内其他 `docs/` 下的文档（`docs/sql.md`、`docs/iceberg-*` 等）为历史调研、对比、深入设计资料，可能领先或滞后于当前主线；如与本目录冲突，**以本目录为评审基线**。
