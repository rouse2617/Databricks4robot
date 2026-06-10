# Proposal — CYB-1218

## Why
`assets.mcap_file_id NOT NULL` 阻止了 `derived_asset` 类型资产创建（跨多 segment 聚合时无单一 MCAP 来源）。

## What Changes

### Modified Capabilities
- **db-schema**: 允许 `derived_asset` 的 `mcap_file_id` 为 NULL，其他类型保持 NOT NULL

## Impact
- **Affected code**: `backend/migrations/036_assets_mcap_file_id_nullable.sql`
- **New APIs**: none
- **Dependencies**: none

## Scope
- **In scope**: DROP NOT NULL + CHECK 约束
- **Out of scope**: audit/event 表的 mcap_file_id 处理、usecase 层更改

## Success Criteria
- [ ] `derived_asset` 可以 INSERT/UPDATE 为 NULL mcap_file_id
- [ ] 非 `derived_asset` 的 mcap_file_id 仍不能为 NULL
