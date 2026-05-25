# Proposal — CYB-1222

## Why
所有资产类型共用通用 `POST /api/v1/assets`，缺少按资产类型的结构化创建端点：
- 调用方需手填 `asset_type`、`parent_asset_id` 等字段
- 不自动写 `asset_relations` 边（split_from / derived_from）
- 没有分类型的 L1-L7 校验

## What Changes

### New Capabilities
- **layered-api**: 4 个按资产类型分层创建端点，自动推断类型、继承父 ID、写 asset_relations 边

## Impact
- **Affected code**: `backend/internal/handlers/asset/`, `backend/internal/usecase/asset/`, `backend/routes/routes.go`
- **New APIs**: 4 POST endpoints (clips, actions, frames, tasks)
- **Dependencies**: none

## Scope
- **In scope**: 4 个 endpoint 的 handler + usecase + routes 注册
- **Out of scope**: 
  - `POST /api/v1/assets/{id}/materialize`（§4.4 物化升级，独立 feature）
  - `raw_mcap` 的 layered API（raw_mcap 通过 upload/finalize 创建）
  - `derived_asset` 的 layered API（需要 multi-source 聚合，独立 feature）
  - Frontend SDK 更新

## Success Criteria
- [ ] 4 个端点成功创建对应类型资产 + asset_relations 边
- [ ] L1-L7 违反返回 422
- [ ] split_method 按决策表决定 split_from vs derived_from
