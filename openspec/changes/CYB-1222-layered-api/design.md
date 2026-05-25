# Design — CYB-1222

## Architecture Context
- **Constraints**: Go 1.25+, Gin router, Postgres 17, 同事务写入 assets + asset_relations + asset_events
- **Goals**: 按资产类型分层创建 API，自动写 asset_relations 边，复用现有 validator
- **Non-Goals**: 不做批量创建、不做 derived_asset multi-source、不做物化升级

## Affected Modules
- `backend/routes/routes.go` — 注册 4 个新路由
- `backend/internal/handlers/asset/handler.go` — 新增 4 个 handler 方法
- `backend/internal/usecase/asset/` — 新增 ChildAssetCreate usecase
- `api/openapi.yaml` — 新 endpoint 契约

## Architecture Decisions

### Decision 1: 复用现有 AssetWriteValidator
- **Approach**: 每个 layered endpoint 内部调用 `AssetWriteValidator.ValidateCreate()` 做 L1-L7 校验
- **Alternative**: 在每个 handler 内单独实现类型校验
- **Rationale**: Validator 已有 L1-L7 完整逻辑，复用避免重复
- **Risk**: Validator 的 L6（mcap_file_id 继承）和 L3（时间窗）仍是 stub，不做校验也不会破坏现有功能

### Decision 2: 单 usecase 方法支持 all 4 types
- **Approach**: 新增 `Usecase.CreateChildAsset()`，接收 type + parent_id + 通用参数，内部 dispatch 到各自逻辑
- **Alternative**: 4 个独立 usecase 方法
- **Rationale**: 4 个类型创建逻辑几乎一样（INSERT assets + asset_relations + events），差异仅在asset_type + 校验
- **Trade-off**: 未来类型专属逻辑可能需要拆开，现阶段一个方法足以

### Decision 3: asset_relations 边类型按 split_method 决策表
- **Approach**: `algo:*` → `derived_from` + run_id metadata；`manual`/`rule:*`/缺失 → `split_from`
- **Alternative**: 始终写 `split_from`，由 caller 额外写边
- **Rationale**: 与 design doc §3.2.1 决策表一致，减少调用方负担
- **Risk**: 决策表与现有 `assets.split_method` + `SplitRunID` 字段有重叠，确保语义一致

### Decision 4: 自定义 request struct 而非复用通用 Create
- **Approach**: 4 个 endpoint 共用精简 request struct（start/end timestamp, metadata, split_method, run_id）
- **Alternative**: 复用现有通用 Create 的完整 request body
- **Rationale**: 通用 request body 含 40+ 字段（含已废弃的 cf_*），对新调用方不友好。精简 struct 只暴露子资产创建相关参数
- **Risk**: 添加精简 struct 后需维护与通用 Create 的字段映射；但 Asset model 是共享的，不会有 drift

## Data Flow

```text
POST /api/v1/assets/{parent_id}/clips
  │
  ├─ 1. Validate parent exists + type=segment (middleware/handler)
  ├─ 2. Validate L1-L7 via AssetWriteValidator
  ├─ 3. Generate new asset_id (8 char)
  ├─ 4. INSERT assets (type=clip, parent_asset_id, timestamps, ...)
  ├─ 5. INSERT asset_relations (parent=new, child=parent, type=split_from|derived_from)
  ├─ 6. INSERT asset_events (type=asset_created)
  └─ 7. Return 201 + created asset
```

## Risks / Trade-offs
| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| split_method 决策表与现有 assets.split_method 字段不匹配 | 边类型选错 | 与现有 usecase.Create 的 splitMethod 处理保持同一逻辑 |
| 现有通用 Create 仍然可用 | 两个路径可能有行为差异 | layered API 作为推荐路径，通用 Create 保持向后兼容 |
