# Proposal — 统一检索 P1：血缘 ES 投影 + 搜索增强

## Why
当前血缘查询是独立端点（`/audit/lineage-search`），不支持在统一搜索中通过血缘过滤资产。需要将 `asset_relations` 投影到 ES，使"某资产的相关资产"成为搜索条件。

## What Changes
- 将 `asset_relations` 投影到 ES，新增 lineage 相关字段
- 在现有搜索 API 中增加 lineage filter（按血缘关联搜索）
- 保持现有 `/audit/lineage-search` 端点兼容

## Scope
- **In scope**: 血缘关系写入 ES、搜索 API 扩展 lineage filter、索引 mapping 变更
- **Out of scope**: Catalog 纳管（P2）、facet/typeahead（P3）、语义检索（P4+）

## Reference Design
`docs/review/unified-search-enhancement.md` (§ 6.2 lineage filter, § 12 P1)
