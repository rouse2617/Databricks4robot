# Proposal — CYB-4011

## Why

DataBrew mcap 资产用 8 位 id（如 `iQw13JU8`），Grace 的 video 用 UUID（如 `019f9893-3456-7376-ae68-30a89227eb46`），两者目前没有显式关联列。已实测可靠的 join key 是 `raw_hash_md5`（内容哈希 = GCS 文件名），`iQw13JU8 → 019f9893-…` 已 1:1 验证。

现状：DataBrew 只隐式持有部分 Grace 外键（vendor/collector/scene/session），缺 Grace video 主键，任何「从 DataBrew 看到对应 Grace video」都只能靠 md5 现查。本变更**预留显式口子**：加一列 `grace_video_id`，端到端可读、可写、可显示、可过滤；**数据填充（回填/增量同步/入库直填）不在本次范围**，列会先为空。

## What Changes

### Modified Capabilities
- **asset-management**: `mcap_files` 新增可空列 `grace_video_id`（Grace video UUID，以 text 存储），沿用已有 flatten-mcap 模式镜像到 `assets`。资产创建请求可选传入 `grace_video_id`；mcap 详情与 `/assets` 查询响应返回该字段；前端 mcap 详情页显示该字段（可复制 + 跳转 Grace），数据为空时显示「未关联」。字段作为**仅过滤**（filter-only）的精确匹配字段暴露，**不做 facet**（Grace UUID 高基数）。

## Impact
- **Affected code**:
  - 迁移：`backend/migrations/<new>.sql`（可空 text 列 + 部分索引，非破坏；off-limits 区，已获发起人授权）
  - 后端：`dbschema/asset_models.go`、`models/asset.go`、`postgres/repos.go`（assets + mcap_files 的 select/scan/insert）、`filter/assets_fields.go`、`searchindex/builder.go`、`handlers/mcap/handler.go`
  - 契约：`api/openapi.yaml`、`docs/review/api-guide.md`、smoke 脚本
  - 前端：`Frontend/src/api/types.ts`、`components/mcap/McapDetailDrawer.tsx`（+ AddFilterPopover 过滤项）
- **New APIs**: 无新端点；在已有 mcap create / mcap get / list、`/queries/run` 响应上增加一个字段。
- **Dependencies**: 无新增。

## Scope
- **In scope**: 加列 + 索引（迁移）；Go 结构体与 repo 读写路径；create 请求开口子；mcap/assets 响应返回；作为 filter-only 精确字段；ES 文档写入；前端类型 + mcap 详情显示 + 过滤项；OpenAPI/api-guide/smoke 同步。
- **Out of scope**（明确本次不做，留后续）:
  - 存量按 md5 **回填**、复用 CYB-3072 的 Grace loop 做**增量同步**、vibecap 入库路径**直填**。
  - B 方案 `assets.grace_id` 指向 Grace segmentation。
  - facet 化 grace_video_id。
  - device_id 两侧不一致核对。

## Success Criteria
- [ ] 迁移在 dev 应用后，`mcap_files` 与 `assets` 均有可空 `grace_video_id` 列 + 部分索引；既有行不受影响（列为 NULL）。
- [ ] mcap create 请求可传 `grace_video_id`，写入后 mcap get / list / `/queries/run` 响应均能读回。
- [ ] `filter=grace_video_id:eq:<uuid>` 能精确命中（filter-only，不出现在 facet 结果里）。
- [ ] 前端 mcap 详情页显示 Grace Video ID，有值时可复制并跳转 `https://grace.cyberorigin.ai/videos/<id>`，无值显示「未关联」。
- [ ] OpenAPI/api-guide/smoke 与实现一致；Tier L 验证通过；dev 部署验证通过。
