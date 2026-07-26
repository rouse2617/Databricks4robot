# Decisions — CYB-4011

## 2026-07-26 — 范围收窄为「只预留口子 + UI 一起做」
- **Context**: 原 Linear description 含存量回填、复用 CYB-3072 增量同步、vibecap 入库直填。用户在会话中明确收窄。
- **Decision**: 本次只加列 + 端到端读写/显示/过滤；回填、增量同步、vibecap 直填、B 方案（assets.grace_id→segmentation）、facet 化 均不做。数据可空，UI 一起做（无值显示「未关联」）。
- **Alternatives**: 一次性做完回填+同步 —— 范围过大，用户否决。
- **Rationale**: 快速打通结构口子，数据填充作为后续独立工作。

## 2026-07-26 — off-limits `backend/migrations/` 授权
- **Context**: 加列需新迁移文件，`backend/migrations/` 属 off-limits。
- **Decision**: 新迁移为可空 text 列 + 部分索引，非破坏、无回填。发起人（ruipeng.huang，本 issue 创建者/负责人）在会话中口头授权动迁移。
- **Rationale**: 非破坏 DDL，风险低；Linear CYB-4011 约束段已记录授权。

## 2026-07-26 — grace_video_id 用 text 且 filter-only
- **Context**: Grace video id 是 UUID 但可能有非规范/遗留值；且为高基数。
- **Decision**: 列类型 `text`（绑定 `nullableText`，参照 `camera_model`），不用 `uuid`；注册为 `exactFieldSpecs` 精确过滤字段，不进任何 facet 路径（planner/asset_facets/query_ir）。
- **Alternatives**: `uuid` 列（对坏值脆，需 per-row 兜底）；facet 化（海量桶，无意义）。
- **Rationale**: 与 flatten 迁移里 device_id/scene_id 被排除 facet 的判断一致。

## 2026-07-26 — SDK 生成文件手工补字段（避免版本漂移噪音）
- **Context**: `sdk/src/cyber_databrew_sdk/_generated/models.py` 由 datamodel-codegen 从 openapi.yaml 生成。本机 `datamodel-codegen` 与仓库 pin 的版本不一致，`make sdk-generate` 因本地 hatchling VCS 版本 tag 无法解析而失败；用非 pin 版本直接跑会重写大量无关 schema（AssetMetadataResponse/TagUpsertRequest/枚举），产生 300+ 行漂移。
- **Decision**: 还原生成文件，只在 `McapFile` 与 `McapCreateFileRequest` 两个生成类中手工追加 `grace_video_id` 字段，风格与相邻 `camera_model` 一致。生成的 `Asset` 类本就缺 `camera_model`（早于 CYB-3715），故不动它。
- **Alternatives**: 提交完整 regen —— 会把历史未同步的无关漂移一并带进本 PR，违反「不清理无关代码」。
- **Rationale**: 保持 PR diff 最小且聚焦；CI `sdk-check-generated` 用 pin 版本重生成校验，`grace_video_id` 会落在同一位置。若 CI 因既有历史漂移报红，属既存问题，另行处理。
