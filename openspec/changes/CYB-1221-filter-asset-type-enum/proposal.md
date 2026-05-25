# Proposal — CYB-1221

## Why
Filter 的 `asset_type` 枚举缺少 `raw_mcap`、`action`、`task` 三个值，且 `frame` 被错误写为 `frame_set`，导致用户无法按这些资产类型筛选搜索结果。

## What Changes

### Modified Capabilities
- **search-filter**: `asset_type` 过滤器枚举值补全为 7 个合法值，修复 `frame_set` → `frame` 拼写错误

## Impact
- **Affected code**: `backend/internal/filter/types.go`
- **New APIs**: none
- **Dependencies**: none

## Scope
- **In scope**: 修正 `SearchableFields["asset_type"].Values`，补全缺失的枚举值
- **Out of scope**: DB CHECK 约束、asset_type 验证逻辑、前端 filter 组件

## Success Criteria
- [ ] `asset_type` filter 支持 7 个值：`raw_mcap`, `segment`, `clip`, `frame`, `action`, `task`, `derived_asset`
- [ ] 前端按任何这 7 个 asset_type 筛选均返回正确结果
- [ ] `frame_set` 不再出现在枚举中
- [ ] `go build ./...` 通过
