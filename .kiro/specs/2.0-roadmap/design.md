# Design — 2.0 Roadmap

## Source of Truth

本 spec 不重复设计，所有技术方案以 `docs/review/` 下的文档为准：

| 领域 | 设计文档 | 关键章节 |
|------|----------|----------|
| 整体架构 | #[[file:docs/review/data-platform-design.md]] | §4 架构、§5 详细设计、§9 实施计划 |
| Outbox Worker | #[[file:docs/review/outbox-worker-design.md]] | §2 架构、§4 cursor、§5 主循环、§9 错误策略、§11 metrics |
| 字段退役 | #[[file:docs/review/data-platform-design.md]] | §5.8.1 API 版本治理与字段退役流程 |
| ES 索引 | #[[file:docs/review/outbox-worker-design.md]] | §6 Doc 投影、§7 ES Sink 与 Mapping |
| 湖仓 | #[[file:docs/review/data-platform-design.md]] | §5.6 数据索引与同步、§5.6.2 两段式入湖、§5.6.4 Iceberg 选型 |
| 事件契约 | #[[file:docs/review/data-platform-design.md]] | §5.9 事件契约与 Schema 演进 |
| PgBouncer | #[[file:docs/review/data-platform-design.md]] | §6.3 数据库接入层 |
| 监控告警 | #[[file:docs/review/data-platform-design.md]] | §8 监控告警 |
| 部署 | #[[file:docs/review/data-platform-design.md]] | §6 服务部署资源 |
| Schema | #[[file:docs/review/schema-reference.md]] | 全表字段定义 |
| API | #[[file:docs/review/api-guide.md]] | 端点 curl 示例 |
| 算法生命周期 | #[[file:docs/review/algo-lifecycle-and-data-model.md]] | 状态机、依赖链 |

## 关键依赖图

```
P0-3 ──► P0-FE-2（lifecycle_state 前端切换）
P0-1 ──► P0-FE-3（新字段暴露到 UI）  ← 已 done
P0-4 ──► P0-T-1（集成测试）──► P0-T-2（CI）
P0-4 ──► P1-1（ES 闸口）
P0-4 ──► P1-8（Worker 独立进程，90 天后）
P1-3 ──► P1-4（Bronze 入湖）──► P1-5（Compact）
P1-6 ──► P1-FE-1（Lakehouse Dashboard）
P1-7 ──► P1-T-4（Schema CI 测试）
P0-6（PgBouncer）独立，可并行
```

## MVP v1 范围

MVP = 最小可用的「2.0 检索闭环」：
1. P0-3（lifecycle_state 主消费）
2. P0-4（Outbox Worker MVP）
3. P1-1（ES 闸口 > 99.9%）
4. P1-2（轻量监控）

MVP 明确延期：Iceberg / Trino / 湖仓 / 训练数据集 UI / PgBouncer。
