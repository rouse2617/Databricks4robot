# 评审文档包 · 数据平台 (Databricks4robot)

本目录是面向方案评审与团队交付的完整文档集合，**自包含、可独立阅读**。

## 阅读顺序

| 序号 | 文档 | 内容 | 推荐读者 |
|------|------|------|----------|
| 1 | [`data-platform-design.md`](./data-platform-design.md) | 数据平台整体方案设计：背景 / 架构 / 核心表 / 字段 / 数据同步 / API / 部署 / 可靠性 / 监控 / 实施计划 / 风险 | **必读**，所有评审人 |
| 2 | [`schema-reference.md`](./schema-reference.md) | PG 全表字段速查 + 上线优先级（Tier 1–5）+ 上线最小 checklist | DBA / 后端 / 数据工程 |
| 3 | [`api-guide.md`](./api-guide.md) | 全部 REST 端点的 curl 示例、错误码参考、典型工作流 | 接 SDK / 前端 / 上下游集成 |
| 4 | [`algo-lifecycle-and-data-model.md`](./algo-lifecycle-and-data-model.md) | 算法生命周期、状态机、依赖解锁、数据模型细节 | 算法 / 后端 |

## 这份文档包覆盖了什么

| 评审议题 | 看哪一篇 |
|----------|----------|
| 整体架构 / 分层 / 组件职责 | 1. data-platform-design §4 |
| Bigtable / Postgres / Spanner 选型 | 1. data-platform-design §5.4 |
| 资产概念定义 / 状态机 | 1. data-platform-design §5.1 + 4. algo-lifecycle |
| 表 schema / 字段定义 / 索引 | 1. data-platform-design §5.2 + 2. schema-reference |
| 数据同步机制（不轮询，outbox + LISTEN/NOTIFY） | 1. data-platform-design §5.6.2 |
| 入湖 / ES 同步流水 | 1. data-platform-design §5.6.2 + §6 |
| API 设计 / 错误码 / 幂等 / 乐观锁 | 1. data-platform-design §5.8 + 3. api-guide |
| 算法接入 / 状态流转 / 依赖编排 | 4. algo-lifecycle |
| 部署形态 / 容量规划 | 1. data-platform-design §6 |
| 故障域 / 冗余 / SLO | 1. data-platform-design §7、§8 |
| 上线计划 / Phase 拆分 / 时间线 | 1. data-platform-design §9 + 2. schema-reference 上线优先级 |
| 风险与未决事项 | 1. data-platform-design §10 |
| 多租户 / PII / GDPR 策略 | 1. data-platform-design §5.2 §10 |

## 这份文档包不覆盖的（评审外议题）

- 多模态数据集物理格式（Daft / Lance）的具体落地 —— Phase 2+ 单独评审
- 训练平台 / 特征平台 / 实验分支管理 —— 独立后续设计
- PII / GDPR 删除链路细则 —— 单独治理文档（待写）
- 上云 Catalog 选型（Polaris / Gravitino / 云原生）的最终决策 —— 上云前 ADR

## 文档维护说明

- 评审通过后，**`api-guide.md` 与 `algo-lifecycle-and-data-model.md` 是项目的长期权威文档**，会持续随代码演进更新。
- `data-platform-design.md` 与 `schema-reference.md` 是阶段性设计快照；当架构发生大变动时会出新版本。
- 仓库内其他 `docs/` 下的文档为历史调研、对比、设计探索资料，未必代表当前结论；如有冲突以本目录文档为准。
