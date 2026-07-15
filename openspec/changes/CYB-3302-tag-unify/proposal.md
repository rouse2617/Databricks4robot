# CYB-3302 — 前端统一 tag.* 拼法（priority/quality）

## Why
facet 侧栏/搜索栏/reducer/URL 层用 `tag.priority`，而 AddFilter/QuickFilters/AGG_KEY_MAP/GROUP_FIELDS 用 `tags_flat.priority` → 两处生成的 chip 字段不同、不去重；AGG_KEY_MAP 与 facet 字段不一致导致 priority/quality facet 拿不到计数。后端对 tag./tags./tags_flat. 三前缀都归一到同一 PG canonical，故统一到 `tag.*` 安全。

## What Changes（纯前端）
- AddFilterPopover：priority/quality key + QUICK_FIELDS → `tag.*`
- QuickFiltersRow：高优先级快捷筛选 → `tag.priority`
- AssetsFacetSidebar：AGG_KEY_MAP、GROUP_FIELDS → `tag.*`；priority/quality CheckboxFacet 加 `counts`

## Impact
- 无后端/schema 变更；过滤走 PG 归一，行为不变。
- scene 的 tags.scene/tags_flat.scene 属另一子情形，本次不动。
- 计数在 UI 显示仍依赖 CYB-3301。
