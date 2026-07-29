# Design — CYB-4271

## Context

从 Grace prod 导入 dev 的 ~10,100 条 mcap（`owner=grace-pull-demo`）只映射了 raw mcap，没带 Grace 侧的转码/派生 URL（forward_stereo/mezzanine/watermark mp4、imu mcap）。mcap **没有通用 update 端点**（`POST` 同 md5 → 409），CYB-4011 的 `grace-video-id` 端点只改一个标量列。本次开一个**窄口**内部端点，把这些派生 URL merge 进既有 mcap 的 `metadata.derived_uris`，并镜像到 raw_mcap 资产的 `files` map。设计目标：与 CYB-4011 `BackfillGraceVideoID` 的事务/镜像/事件模式完全对齐，**零 schema 变更**（复用既有 JSONB 列）。

## Decisions

### D1 — 落在既有 JSONB，无 migration
- **决定**：派生 URL 写进 `mcap_files.metadata`（既有 JSONB）的 `derived_uris` 子对象；镜像到 `assets` 的 `files`（既有 JSONB）。不加列、不加迁移。
- **理由**：`metadata`/`files` 已在读写路径完整 round-trip（`McapFileRepo.Get/Set` marshal 整个 `metadata`；`AssetRepo.Set` marshal `FilesJSON`）。避免 off-limits `backend/migrations/` 与部署顺序风险。
- **备选**：为每个 URL 加 flatten 列（如 CYB-4011）——重、且这些 URL 不需要 facet/精确过滤，放弃。

### D2 — merge 语义（只增不覆盖），幂等
- **决定**：读出既有 `metadata.derived_uris`（若有），逐 key 覆盖传入的非空值，其它 metadata 键（`env`/`task`/…）原样保留；merge 前后完全相同 → 不写库、不发事件、直接 200。
- **理由**：回填脚本可安全重跑；不同来源分批补 URL 不会互相清空。
- **实现**：`changed` 标志比较传入值与既有值，决定是否进事务。

### D3 — 镜像写 `FilesJSON`（不是 `Files`）
- **决定**：镜像时同时写 `a.FilesJSON[k]` 与 `a.Files[k]`，但**事实源是 `FilesJSON`**。
- **理由**：`AssetRepo.Set` 的 `bindAssetJSONAndRefs` 只有在 `FilesJSON==nil && len(Files)>0` 时才从 `Files` 派生；而 `Get` 之后 `FilesJSON` 非 nil（由 `filesBytes` unmarshal 而来），因此 `Set` 实际 marshal 的是 `FilesJSON`。若只改 `Files` 会**静默丢失**。`Files` 同步只为响应体/下游消费者可读。

### D4 — 内部 admin 路由，与 grace-video-id 同组
- **决定**：`PATCH /api/v1/internal/mcap-files/:id/derived-uris`，挂在 `internalMcap`（`adminRoutesEnabled` gated，`adminAuth`）。
- **理由**：回填是运维动作，非公共面；与 CYB-4011 同组保持一致。dev 下 `ADMIN_TOKEN` 未设 → `AdminTokenAuth` 回退 `StaticTokenAuth(databrewToken)`，故 smoke 用 `X-Databrew-Token` 即可命中。
- **字段**：`raw_mcap_uri`/`forward_stereo_mp4`/`mezzanine_mp4`/`watermark_mp4`/`imu_mcap_uri`，全部可选，至少一个非空，否则 400。

### D5 — 事件与 ES
- **决定**：merge 有实际变更时发 `asset_updated`（payload 带 `asset_id` + `derived_uris`），触发 ES `Build(assetID)`。派生 URL 是否进 ES 检索面**不额外扩**——沿用 builder 既有对 `files`/`metadata` 的处理；本次不加新的检索字段。
- **理由**：目标是资产页「文件」可见 + 一致性重建，不是让 URL 可被搜索。

## Data Flow

```
PATCH /internal/mcap-files/{id}/derived-uris  { forward_stereo_mp4, mezzanine_mp4, ... }
        │
        ▼ (collect non-empty, ≥1 else 400)
   McapFileRepo.Get ──▶ merge into f.Metadata["derived_uris"] (siblings preserved)
        │                         │ changed? no → 200, no write
        │ withTx                  ▼ yes
        ├── McapFileRepo.Set ─────────▶ mcap_files.metadata.derived_uris
        ├── AssetRepo.Get(id) ──▶ merge into a.FilesJSON (+ a.Files) ──▶ AssetRepo.Set ──▶ assets.files
        └── AssetEventRepo.Append(asset_updated) ──▶ outbox ──▶ ES Build(assetID)
        ▼
   200 McapFile (metadata.derived_uris populated)

Backfill script (one-shot, read-only Grace): md5 → Grace storage_meta → PATCH endpoint
```

## Migration & Risk

| 项 | 说明 | 风险 | 缓解 |
|---|---|---|---|
| 无 migration | 复用既有 `metadata`/`files` JSONB | 低 | 无 schema 变更 |
| merge 覆盖 metadata | 只写 `derived_uris` 子键 | 中（误删 sibling） | 读出全 metadata 再改子对象；单测断言 `env` 保留 |
| 镜像丢失 | 只改 `Files` 不改 `FilesJSON` 会静默丢 | 中 | D3：写 `FilesJSON`；单测断言 `FilesJSON` 命中 |
| 内部路由鉴权 | admin-gated | 低 | 同 CYB-4011；dev 回退 databrew token |
| 回填脚本 Grace prod key | 只读 Grace | 中（凭据泄露） | key 不回显/不入文件/不入仓库；只写 dev |

## Verification Tier
**Tier M** — 触及 `backend/internal/handlers` + `routes` + OpenAPI 契约 + smoke；无 migration、无前端路由改动。含契约同步与 dev 部署自测（PATCH → get 读回 + 资产页文件可见 + 幂等）。
