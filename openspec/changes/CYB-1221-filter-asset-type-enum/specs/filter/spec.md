## MODIFIED Requirements

### Requirement: asset_type 过滤器枚举完整性
- **Before**: The system SHALL accept `segment`, `clip`, `frame_set`, `derived_asset` as valid `asset_type` filter values.
- **After**: The system SHALL accept `raw_mcap`, `segment`, `clip`, `frame`, `action`, `task`, `derived_asset` as valid `asset_type` filter values.
- **Reason**: 缺失的枚举值导致无法按 `raw_mcap`/`action`/`task` 筛选；`frame_set` 与实际存储的 `frame` 不匹配，筛选永远匹配不到 frame 类型资产。

**Priority**: P0 (Critical)
**Rationale**: 用户完全无法按这三种资产类型筛选搜索结果，属于功能缺陷。

#### Scenario: 按 raw_mcap 筛选
- **Given** 存在 asset_type=raw_mcap 的资产
- **When** 用户查询 `asset_type:eq:raw_mcap`
- **Then** 返回所有 raw_mcap 资产，不报错

#### Scenario: 按 frame 筛选（修复后）
- **Given** 存在 asset_type=frame 的资产
- **When** 用户查询 `asset_type:eq:frame`
- **Then** 返回所有 frame 资产，不报错

#### Scenario: 使用旧值 frame_set 被拒绝
- **Given** filter 已修复为 `frame`
- **When** 用户查询 `asset_type:eq:frame_set`
- **Then** 返回 422 错误，提示非法枚举值
