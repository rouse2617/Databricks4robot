# ADR-004:不 fork DataHub / OpenMetadata

- **Status**: Accepted
- **Date**: 2026-04-22
- **Related**: [06-borrowed-patterns.md](../06-borrowed-patterns.md)

---

## Context

DataHub 和 OpenMetadata 是业界最知名的两个开源元数据系统。团队在调研数据平台选型时,自然想到"能否 fork / 直接用"。

但:
- 它们的 **核心 Entity 是"表 / 字段 / Dashboard"** —— 为 DE / Analytics 场景设计
- 我们的 **核心 Entity 是"MCAP 视频 segment"** —— 机器人/CV 场景
- 它们自带 50+ 数据源 connector(Snowflake/BigQuery/Kafka/dbt/...),**我们只有 1 个数据源(MCAP)**
- 它们假设"数据已经在上游数据库 / 数据湖里了",**我们负责把 MCAP 文件变成结构化资产**

## Decision

**不 fork**,也**不在 Phase 0/1 作为 sidecar 集成**。

改为:
1. **抄 10 个核心设计模式**(URN / Aspect / MCE-MCL / Ingestion / DQ / Lineage / Classification-Tag-Glossary / Activity / Actions / Stateful),参见 [06-borrowed-patterns.md](../06-borrowed-patterns.md)
2. **按需直接用少量组件**:
   - OpenMetadata 的 `ingestion-framework`(Source/Processor/Sink 骨架)
   - OpenMetadata 的 `data-quality` 测试引擎
3. **Phase 2+ 评估**是否把它们作为"表 / 字段"元数据(如 BigQuery 表)的辅助目录(可选)

## Alternatives Considered

### A. Fork DataHub 作为平台底座

- 🔴 Entity 模型是"表 / dataset",改成"MCAP segment"要改 30% 以上代码
- 🔴 Pegasus schema 语言陡峭,团队学习成本高
- 🔴 GMS / metadata-service 用 Java/Scala,团队以 Go/Python 为主
- 🔴 后续升级与我们魔改的冲突难调和

### B. Fork OpenMetadata 作为平台底座

- 🟡 比 DataHub 稍轻
- 🔴 依然是"数据库表"为核心,Entity 类型扩展仍需改后端
- 🔴 JSON Schema-first,与我们用 Bigtable 列族模型不符
- 🔴 维护成本高

### C. Sidecar 集成(平台自研 + 同步 metadata 到 OpenMetadata / DataHub)

- 🟡 Phase 2+ 可以考虑,为 DE 团队提供传统元数据视图
- 🔴 Phase 0/1 复杂度翻倍,没必要

### D. 不看这些项目,全自研

- 🔴 浪费;人家总结的 10 个模式是业界最佳实践,不用就是自断双腿

## Consequences

### Positive
- ✅ 团队精力聚焦在 `data4cyber` 核心业务(MCAP / Segment / 算法用户 SDK)
- ✅ 吸收开源精华,设计档次对齐业界一线
- ✅ 避免深度耦合,架构演进自由

### Negative
- ⚠️ 需要**自觉**去借鉴(不强制),可能漏掉一些好的模式
- ⚠️ 如果以后要对外集成(BI / DE 工具),没有现成的数据目录接入

### Mitigations
- 把"偷师清单"([06-borrowed-patterns.md](../06-borrowed-patterns.md))作为必读文档
- Phase 2 评估:如果确实需要对外数据目录能力,再启动 OpenMetadata sidecar 集成(1-2 人月)

## Acceptance Criteria

- 项目代码中不存在 datahub / openmetadata 的 runtime 依赖
- 设计文档中引用了至少 5 个来自这两家的模式(URN / Aspect / MCE-MCL / Ingestion / DQ)
- Phase 0 结束时完成团队内部 2 次 share(OpenMetadata 1 次 + DataHub 1 次)

## References

- DataHub: https://github.com/datahub-project/datahub
- OpenMetadata: https://github.com/open-metadata/OpenMetadata
- 偷师清单: [06-borrowed-patterns.md](../06-borrowed-patterns.md)
