# 评审文档包 · 数据平台 (Databricks4robot)

本目录是面向方案评审与团队交付的完整文档集合，**自包含、可独立阅读**。

## 当前架构基线

> **当前位置：1.0**（仅 PostgreSQL 单库 + Backend，2.0 尚未启动）。
>
> 本文档包同时描述 **Current（1.0 已落地形态）** 与 **Target（2.0 / 3.x 演进目标）** 两套口径；遇到冲突时，本目录所有文档都以下面这条主线为准：
>
> - 主库：PostgreSQL（不再使用 Bigtable）
> - 当前态：标量列 + `asset_tags` / `asset_algo_latest` 投影表 + `asset_events` 统一事件表 —— **1.0 已建已用**
> - `asset_events` 是事件主线起点；当前分支以 **CDC/WAL** 作为下游同步机制（ES / Iceberg）
>
> 任何文档片段如果还在描述 "cf_algo 是主路径"、"Phase 1 切 Bigtable" 之类的形态，都属于**历史路径说明**，不是当前主架构。

## 阅读顺序

| 序号 | 文档 | 内容 | 推荐读者 |
|------|------|------|----------|
| 1 | [`data-platform-design.md`](https://www.feishu.cn/wiki/QiNWwqLlWinHQpkf9Pbcy0pfniB) | 数据平台整体方案设计：背景 / 架构 / 核心表 / 字段 / 数据同步 / API / 部署 / 可靠性 / 监控 / 实施计划 / 风险 | **必读**，所有评审人 |
| 2 | [`schema-reference.md`](https://www.feishu.cn/wiki/BUvcwpQeAiPWNtkLpKecDwkcnxd) | PG 全表字段速查 + 上线优先级（Tier 1–5）+ 上线最小 checklist | DBA / 后端 / 数据工程 |
| 3 | [`api-guide.md`](https://www.feishu.cn/wiki/OEG4wYA48i3Kvpk0N1XccwW8nqe) | 全部 REST 端点的 curl 示例、错误码参考、典型工作流；含**新旧字段命名对照表**（lifecycle_state / asset_type / duration_ms） | 接 SDK / 前端 / 上下游集成 |
| 4 | [`algo-lifecycle-and-data-model.md`](https://www.feishu.cn/wiki/EoYowiw4ji5BO0kxODgce1hgnPh) | 算法生命周期、状态机、依赖解锁、数据模型细节（主线：`asset_algo_latest + asset_events`） | 算法 / 后端 |
| 5 | [`use-cases.md`](https://www.feishu.cn/wiki/RqiIwqJGAigsM9k2AZecJ4punce) | 全部角色 × 业务域 use case 矩阵，约 65 项端点 + 关键约束 + 开发优先级；**🟢/⚪** 标注现行路由 vs 目标态 | 前端 / SDK / 后端 |
| 6 | [`eval-metrics-design.md`](https://www.feishu.cn/wiki/DOTCwUSOPiapEykJeoxcNctSnwf) | Eval/Metrics 专项设计：表结构、事件、API、ES/Iceberg、Phase 1.5 落地 | 后端 / 数据工程 / 算法 |
| 7 | [`asset-events-bronze-design.md`](https://www.feishu.cn/wiki/CZ2cwoaGTiH9CIkror7c962In4g) | 长期方案：`asset_events` 通过 WAL/CDC 同步到 Iceberg Bronze，ES 从 current-state tables 的 CDC 重建文档 | 后端 / 数据工程 |
| 8 | [`cdc-rollout-plan.md`](https://www.feishu.cn/wiki/FQQDwYifhi7DgBkGw7VcKkmwn0b) | CDC/WAL 同步实施方案（Bronze + ES current-state），测试、上线、回滚计划 | 后端 / 数据工程 / 运维 |
| 9 | [`pure-cdc-go-live-runbook.md`](https://www.feishu.cn/wiki/AiCxwlDSfiOiUbkrWumcVgKXnnc) | **上线执行手册**：Pure CDC 单路径决策、Go/No-Go、压测/对账、告警、灰度、回滚、值班排障 | 后端 / 数据工程 / SRE |
| 10 | [`grace-migration-notes.md`](https://www.feishu.cn/wiki/BrCHw7NrZiOVGik8lfCcl555nhh) | **cyber-grace**（`grace_videos`）与本平台 `assets` / `mcap_files` / 算法投影的字段与幂等对照 | 集成 / 后端 / 数据工程 |
| 11 | [`asset-events-thinning-plan.md`](https://www.feishu.cn/wiki/HkjKwNggdiB5TBkgktXcSWmBn0c) | **裁决文档**：`asset_events` 该保留什么、该砍什么、代码与文档怎么收窄 | 架构 / 后端 / 数据工程 |
| 12 | [`sql.md`](https://www.feishu.cn/wiki/HouqwVou0ijHUzkaiSYchivnnDh) | **Schema companion**：保留 DDL/字段 section 编号，给 `schemas/pg-phase0.sql`、历史注释与 review 使用 | 后端 / DBA |
| 13 | [`query-platform-design.md`](https://www.feishu.cn/wiki/SLTFwOEOAi8uxDkoMzPciswfnNd) | **新提案**：统一查询内核（`JSON Query IR + Planner + Executors`），承载 saved query / slice / export / scenario test | 架构 / 后端 / 前端 / SDK |
| 14 | [`3.0-multimodal-design.md`](https://www.feishu.cn/wiki/EkaOw77wtiX00BkwVh3cHvzsn4e) | **未来草稿**：3.x 多模态检索 / Lance 路线，未进入当前 1.0/2.0 基线 | 架构 / 算法 |
| 15 | [`doc-alignment-pure-cdc-checklist.md`](https://www.feishu.cn/wiki/SQtawoaRyiFfUKkMw4WcSdKUnyh) | **文档收口执行清单**：逐文件统一到 Pure CDC 口径（冲突点、替换建议、验收标准） | 架构 / 后端 / 数据工程 / SRE |
| 16 | [`outbox-worker-design.md`](https://www.feishu.cn/wiki/YDQPwfT53i3lXZkoj3mcFv7An2c) / [`outbox-test-plan.md`](https://www.feishu.cn/wiki/N3BCw4SbDirpjpkbfbAclKLInUd) | **历史方案参考**：Outbox Worker 的历史设计与测试手册；当前生产默认路径请以 Pure CDC 文档为准 | 架构 / 后端（回溯排障） |

## 这份文档包覆盖了什么

| 评审议题 | 看哪一篇 |
|----------|----------|
| 整体架构 / 分层 / 组件职责 | 1. data-platform-design §4 |
| Bigtable / Postgres / Spanner 选型 | 1. data-platform-design §5.4 |
| 资产概念定义 / 状态机 | 1. data-platform-design §5.1 + 4. algo-lifecycle |
| 业务模型 `mcap → seg → action` / 标注承载矩阵 / 时间区间检索 | 1. data-platform-design §5.1 + §5.2.15 |
| 表 schema / 字段定义 / 索引 | 1. data-platform-design §5.2 + 2. schema-reference |
| 数据同步机制（CDC/WAL） | 1. data-platform-design §5.6.2 + cdc-rollout-plan |
| 事件契约 / payload schema 演进 | 1. data-platform-design §5.9 |
| `asset_events` 的边界该不该收窄 | 11. asset-events-thinning-plan |
| 统一查询层 / saved query / slice / 多引擎 planner | 13. query-platform-design |
| 一致性语义 / 重试 / DLQ / 对账 / 回放 | 1. data-platform-design §5.10 |
| 入湖 / ES 同步流水 | 1. data-platform-design §5.6.2 + §6 |
| API 设计 / 错误码 / 幂等 / 乐观锁 | 1. data-platform-design §5.8 + 3. api-guide |
| Grace 生产库 ↔ 本平台字段 / ingest 幂等 | 7. grace-migration-notes |
| API 版本治理 / 字段退役流程 | 1. data-platform-design §5.8.1 + 3. api-guide 顶部对照表 |
| 算法接入 / 状态流转 / 依赖编排 | 4. algo-lifecycle |
| 部署形态 | 1. data-platform-design §6 |
| 故障域 / 冗余 / SLO | 1. data-platform-design §7、§8 |
| 安全 / 权限 / 删除 SLA | 1. data-platform-design §7.1（密钥与数据分级由独立运维 / 合规手册承载）|
| 上线计划 / Phase 拆分 | 1. data-platform-design §9 + 2. schema-reference 上线优先级 |
| Phase Gate（验收 + 回滚） | 1. data-platform-design §9.1 |
| 风险与未决事项（含 owner / 关闭标准） | 1. data-platform-design §10.1 |
| PII / GDPR 策略 | 1. data-platform-design §7.1.2 |

## 这份文档包不覆盖的（评审外议题）

- 容量预算 / 1x/3x/10x 资源模型 —— 待 2.0 真流量上线后由独立 Capacity ADR 承载
- RTO/RPO 量化 / 容灾演练计划 —— 由独立 SRE 文档承载（需先有客户 SLA 输入）
- oncall runbook / 告警处理流程 —— 由运维手册承载
- RACI / 团队职责边界 —— 团队稳定后单独 ownership map
- PG / ES / Iceberg 物理调优细节（索引 / 分区 / compact 节奏） —— 各自上线前发实施 ADR
- 多模态数据集物理格式（Daft / Lance）的具体落地 —— Phase 2+ 单独评审
- 训练平台 / 特征平台 / 实验分支管理 —— 独立后续设计
- PII / GDPR 删除链路细则 —— 单独治理文档（待写）
- 密钥管理 / 凭据轮换具体方案 —— 由独立运维手册承载（K8s Secret / Workload Identity / Vault 选型）
- 数据分级（L1–L4）与脱敏链路细节 —— 由独立合规文档承载
- 上云 Catalog 选型（Polaris / Gravitino / 云原生）的最终决策 —— 上云前 ADR

## 文档维护说明

本目录文档定位与生命周期：

| 文档 | 类型 | 维护节奏 | 权威性 |
|------|------|----------|--------|
| `data-platform-design.md` | 阶段性设计快照（1.0 → 3.x 整体演进） | 架构发生重大变化时整篇出新版本 | 评审基线，**架构主口径**以这篇为准 |
| `schema-reference.md` | 字段速查 + 上线优先级 | 表 / 字段变化时跟随更新 | DDL / 字段定义以 [`schemas/pg-phase0.sql`](../../schemas/pg-phase0.sql) 为最终源；本文档为人类可读视图 |
| `api-guide.md` | API 使用指南 + curl 示例 | 长期维护，随后端 API 演进同步更新 | API 行为最终以 [`api/openapi.yaml`](../../api/openapi.yaml) 为源 |
| `algo-lifecycle-and-data-model.md` | 算法子领域设计 | 长期维护，随算法生命周期 / 状态机演进同步更新 | 与 `data-platform-design.md` 冲突时以**整体设计文档为准** |
| `query-platform-design.md` | 查询内核专项设计 | 查询协议 / planner / 执行器边界变化时更新 | 与 `api-guide.md` / `openapi.yaml` 的查询接口冲突时，以实现中的 API 契约为准 |
| `sql.md` | DDL 伴随文档 / 历史引用兼容层 | 仅在表结构与 section 编号变化时更新 | 不再承载 ES / Iceberg / 同步链路主设计；这些内容以主设计文档为准 |
| `3.0-multimodal-design.md` | 未来方向草稿 | 进入 3.x 立项后再提升为主文或 ADR | 当前不参与 1.0/2.0 基线裁决 |

仓库内 `docs/archive/` 及其他 review 外文档多为历史调研、对比或深入草稿；如与本目录冲突，**以本目录为评审基线**。
