# Asset Management Spec Delta — CYB-4011

## ADDED Requirements

### Requirement: mcap 资产携带可选的 Grace video id

The system SHALL persist an optional `grace_video_id` (Grace video UUID, stored as text) on `mcap_files` and mirror it onto the derived `assets` row, exposing it for read, write-on-create, and exact filtering. The field MAY be empty (unpopulated) — this change reserves the interface without backfilling data.

**Priority**: P2 (Medium)
**Rationale**: DataBrew mcap（8 位 id）与 Grace video（UUID）目前无显式关联列，只能靠 `raw_hash_md5` 现查。预留显式列使前端可显示、API 可读写、查询可过滤，为后续回填/同步打通口子。

#### Scenario: 既有资产不受影响（列为空）
- **Given** 迁移应用前已存在的 mcap 资产
- **When** 迁移应用后读取该资产
- **Then** `grace_video_id` 为空（NULL / 空字符串），其余字段与行为不变

#### Scenario: 创建时写入并读回
- **Given** 创建 mcap 文件的请求带 `grace_video_id`（一个 Grace video UUID）
- **When** 该 mcap 文件与其 raw_mcap 资产被创建
- **Then** mcap get、mcap list、`/queries/run` 的资产响应都返回相同的 `grace_video_id`

#### Scenario: 按 grace_video_id 精确过滤
- **Given** 若干资产带有不同的 `grace_video_id`
- **When** 查询使用 `filter=grace_video_id:eq:<uuid>`
- **Then** 仅返回该 `grace_video_id` 匹配的资产

#### Scenario: grace_video_id 不作为 facet 暴露
- **Given** 资产集合带有多个不同的 `grace_video_id`（高基数）
- **When** 请求 facet 聚合
- **Then** `grace_video_id` 不出现在可用 facet 字段中（仅支持精确过滤）

#### Scenario: 前端显示与占位
- **Given** 用户打开某 mcap 的详情（或资产详情页）
- **When** 该资产有 `grace_video_id`
- **Then** 页面显示该 id 且可复制（不提供跳转 Grace 的外链）
- **When** 该资产无 `grace_video_id`
- **Then** 页面显示「未关联」占位
