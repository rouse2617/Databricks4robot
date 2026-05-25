## MODIFIED Requirements

### Requirement: assets.mcap_file_id 非空约束
- **Before**: The system SHALL require `mcap_file_id` for all asset types.
- **After**: The system SHALL require `mcap_file_id` for all asset types except `derived_asset`.
- **Reason**: derived_asset 跨多 segment 聚合时无单一 MCAP 来源，必须允许 NULL。

**Priority**: P0 (Critical)
**Rationale**: 这是实现 L6 derived_asset 的阻塞性前置条件。

#### Scenario: derived_asset 允许 NULL mcap_file_id
- **Given** asset_type = 'derived_asset'
- **When** INSERT/SET `mcap_file_id = NULL`
- **Then** 操作成功

#### Scenario: 非 derived_asset 仍要求 mcap_file_id
- **Given** asset_type = 'segment' (或其他非 derived_asset)
- **When** INSERT/SET `mcap_file_id = NULL`
- **Then** 返回 CHECK 约束违反错误
