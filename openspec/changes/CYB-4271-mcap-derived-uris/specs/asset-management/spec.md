# Asset Management Spec Delta — CYB-4271

## ADDED Requirements

### Requirement: mcap 派生 URL 可通过内部端点回填

The system SHALL provide an internal admin-gated endpoint that merges Grace-derived asset URLs (transcoded mp4 variants and derived mcap URIs) into an existing mcap file's `metadata.derived_uris` object, mirrors them onto the derived `raw_mcap` asset's `files` map, and emits an `asset_updated` event so the search document reindexes. The merge SHALL be additive: it MUST NOT drop or overwrite unrelated `metadata` keys, and it MUST be idempotent.

**Priority**: P2 (Medium)
**Rationale**: 从 Grace prod 导入 dev 的 mcap（`owner=grace-pull-demo`，~10,100 条）只映射了 raw mcap，未带 Grace 侧的 forward_stereo / mezzanine / watermark mp4 与 imu mcap 等派生 URL。mcap 没有通用 update 端点（`POST` 同 md5 → 409），CYB-4011 的 grace-video-id 端点只能改一个字段。本需求开一个窄口，使已导入的 mcap 能补上派生 URL，前端资产页与下游可见转码产物。

#### Scenario: 合并派生 URL 到 metadata.derived_uris
- **Given** 一个已存在的 mcap（其 `metadata` 已含若干无关键，如 `env`、`task`）
- **When** 调用 `PATCH /api/v1/internal/mcap-files/{id}/derived-uris`，body 带 `forward_stereo_mp4`、`mezzanine_mp4`、`watermark_mp4`、`raw_mcap_uri`、`imu_mcap_uri` 中的一个或多个
- **Then** mcap 的 `metadata.derived_uris` 出现这些 key 对应的 URL，且 `metadata` 原有无关键（`env`、`task` 等）保持不变

#### Scenario: 镜像到 raw_mcap 资产的 files map 并重建索引
- **Given** 该 mcap 对应的 `raw_mcap` 资产（`asset_id == mcap_file_id`）未被删除
- **When** 端点成功合并派生 URL
- **Then** 该资产的 `files` map 含对应 URL（资产详情页「文件」可见），且一条 `asset_updated` 事件已入 outbox（ES 文档随后重建）

#### Scenario: 幂等
- **Given** 同一组派生 URL 已通过该端点写入某 mcap
- **When** 用相同 body 再次调用该端点
- **Then** 结果稳定（`metadata.derived_uris` 与 `files` 不变），不产生冲突错误

#### Scenario: 缺失校验与不存在
- **Given** 调用该端点
- **When** body 为空 / 不含任何可识别的派生 URL key
- **Then** 返回 400（invalid argument）
- **When** 路径 `id` 对应的 mcap 不存在
- **Then** 返回 404

#### Scenario: 不改动无关资产字段
- **Given** 目标 mcap 与资产已有 `grace_video_id`、tags、其它 files 键
- **When** 端点合并派生 URL
- **Then** 仅新增/更新派生 URL 对应的 metadata/files 键，`grace_video_id`、tags、其它 files 键均不受影响
