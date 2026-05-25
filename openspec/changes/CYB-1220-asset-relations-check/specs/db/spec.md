## ADDED Requirements

### Requirement: asset_relations CHECK 约束
The system SHALL enforce that `asset_relations.relation_type` is one of exactly 6 allowed values.

**Priority**: P0 (Critical)
**Rationale**: 无约束下可写入任意值，导致 lineage 查询不可靠、数据损坏。

#### Scenario: 写入合法 relation_type 成功
- **Given** asset_relations 表期望 6 种合法边类型
- **When** 插入 `split_from`, `derived_from`, `contains`, `sampled_from`, `merged_from`, `revision_of` 之一
- **Then** INSERT 成功

#### Scenario: 写入非法 relation_type 被拒绝
- **Given** asset_relations 表有 CHECK 约束
- **When** 插入 `unknown_type` 等非法值
- **Then** INSERT 失败，返回 CHECK 约束违反错误
