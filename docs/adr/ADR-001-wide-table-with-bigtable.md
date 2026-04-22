# ADR-001:用 Bigtable 宽表作为元数据底座

- **Status**: Accepted
- **Date**: 2026-04-22
- **Deciders**: 架构组
- **Related**: [03-data-model-wide-table.md](../03-data-model-wide-table.md) / [ADR-007](ADR-007-write-path-and-event-contract.md) / [ADR-009](ADR-009-phase1-migration-plan.md)

---

## Context

我们要建设 `data4cyber` 资产数据平台,用户明确要求:
- **宽表设计**,`asset_id` 作为主键
- 支持**频繁新增列**(业务 tag / 算法产物不断演进)
- 支持**多版本**(asset 变更历史)
- 支持**亿级规模**(1 年后 500 万,3 年后 1 亿+ asset;tag 总量 50 亿+)
- 云锁定 GCP

## Decision

**元数据主表采用 Cloud Bigtable,按"列族 ≈ aspect"的方式组织。`asset_id` 是对外的**逻辑主键**(URN 的 `<key>`),物理 row key 另行设计以携带稳定业务属性并打散写入热点。**

- **逻辑主键 (对外)**:`asset_id`(UUIDv4)
- **物理 row key (Bigtable)**:详见 [03-data-model-wide-table.md §2](../03-data-model-wide-table.md)
  - Phase 0 (PG 模拟):`assets.asset_id` 就是表主键
  - Phase 1 Bigtable:`v0#<tenant>#<project>#<asset_uuid>` → `v1#<salt2>#<tenant>#<reverse_ts_ms_19>#<project>#<asset_uuid>`
- **`asset_id` 点查**:Phase 1 起走**二级索引表 `asset_locator`**(`<asset_uuid>` → `main_row_key`),再 `GetRow` 主表。这一跳对用户(SDK / UI)**完全透明**。
- **配套索引表**:`asset_locator`(ID 点查入口)+ `assets_by_project`(按 project 扫最近)

Phase 0 先用 Cloud SQL for Postgres 的 JSONB 字段模拟同构(**8 个 JSONB 列 + `asset_events` 独立表** ≈ Bigtable 的 **9 个列族**,其中 cf:event 单独为表),Phase 1 迁移到真 Bigtable。

## Alternatives Considered

| 方案 | 优点 | 为什么不选 |
| --- | --- | --- |
| **Cloud Spanner** | SQL 强、全球一致 | 成本高 10x;百亿行点查不如 Bigtable;我们不需要跨行事务 |
| **Cloud SQL Postgres**(长期) | JSONB 灵活、SQL 丰富 | **百亿行写放大严重**;GIN 索引更新代价大 |
| **Firestore** | Managed 文档库 | 不擅长宽列;查询受限;计费复杂 |
| **BigQuery as OLTP** | 同一技术栈 | 非 OLTP 设计;单行更新贵 |
| **HBase(自运维)** | 与 Bigtable 模型一致,开源 | 运维成本大;GCP 原生已有 Bigtable |
| **Aliyun Lindorm** | All-in-one | 需锁定阿里云,与"GCP 全栈"冲突 |
| **MongoDB** | 文档灵活 | 扩展性不如 Bigtable;GCP 非一等公民 |

## Consequences

### Positive
- ✅ **原生多版本 / 单行原子**,业务需求零成本满足
- ✅ **稀疏列 → 加列零成本**,业务演进无痛
- ✅ **水平扩展线性**,从几万扩到十亿行无停机
- ✅ **点查 P99 < 10ms**,算法用户体验好
- ✅ 与 GCP IAM / Monitoring / BigQuery 原生打通

### Negative
- ⚠️ **学习曲线**:团队对 Bigtable 的 schema 设计(row key / CF)需要培训
- ⚠️ **无 SQL**:分析 / 聚合必须走 BigQuery External Table
- ⚠️ **无跨行事务**:需要业务层保证(MCE/MCL 补)
- ⚠️ **搜索要另建索引**:Tag 组合过滤必须 CDC → OpenSearch

### Mitigations
- Phase 0 用 PG JSONB 模拟,团队先熟悉列族思想,再切 Bigtable(降低风险)
- 所有复杂查询强制走索引副本,主表只做 CRUD 点查
- SDK 封装多行批量操作,业务层不直接访问 Bigtable
- **`asset_locator` / `assets_by_project` 的跨行写一致性**由 asset-service 负责(详见 [ADR-007 §4](ADR-007-write-path-and-event-contract.md#4-多表写原子性phase-1-bigtable-起))

## Acceptance Criteria

- Phase 0 PG 宽表上线,跑 500k 条 asset 不出现性能问题
- Phase 1 Bigtable 切换完成:**locator 点查 + 主表 GetRow 端到端 P99 < 50ms**(含二级索引一跳)
- 每次加新列业务代码零变更(只改 schema registry)
- `asset_locator` 无孤儿率(> 1 分钟未补齐的主表行 < 1/百万)

## References

- Google Bigtable Schema Design: https://cloud.google.com/bigtable/docs/schema-design
- Instagram wide-row pattern(类似思路):https://instagram-engineering.com/
