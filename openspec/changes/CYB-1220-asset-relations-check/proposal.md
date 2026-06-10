# Proposal — CYB-1220

## Why
`asset_relations` 表没有 CHECK 约束，允许写入 `relation_type` 非法值，破坏数据完整性。

## What Changes

### New Capabilities
- **db-constraint**: 在 `asset_relations` 表添加 `chk_relation_type` CHECK 约束限制 6 种合法边类型

## Impact
- **Affected code**: `backend/migrations/035_asset_relations_check_constraint.sql`
- **New APIs**: none
- **Dependencies**: none

## Scope
- **In scope**: CHECK 约束 + 现网脏数据预检
- **Out of scope**: asset_relations 其他列约束、metadata JSONB 列、VIEW 创建

## Success Criteria
- [ ] `asset_relations` 表写入非法 `relation_type` 返回 422/DB 错误
- [ ] 6 种合法值均正常写入
