# Proposal — CYB-4271

## Why

从 Grace prod 导入 dev 的 mcap 资产（`owner=grace-pull-demo`，~10,100 条）只映射了 raw mcap（`gcs_path=gs://.../raw/<md5>.mcap`），没带 Grace 侧的转码/派生 URL：

- `storage_meta.gcs.algo_inputs.mcap.uri` → **forward_stereo mp4**（`gs://.../forward_stereo/<md5>.mp4`）
- `storage_meta.gcs.annot_inputs.right.uri` → **mezzanine 540p mp4**
- `storage_meta.aliyun...watermark` → **watermark 540p mp4**（oss）
- `storage_meta.gcs.imu` → imu mcap（部分有）

mcap **没有通用 update 端点**（只有 `POST`/`GET`/`bytes`；CYB-4011 的 `PATCH /internal/mcap-files/:id/grace-video-id` 只能改一个字段），re-POST 同 md5 → 409。所以已导入的 mcap 无法补这些 URL，前端/下游拿不到转码 mp4。

## What Changes

### Modified Capabilities
- **asset-management**: 新增内部 admin 端点，把 Grace 派生 URL（forward_stereo/mezzanine/watermark mp4、raw/imu mcap）merge 进既有 mcap 的 `metadata.derived_uris`（不覆盖其它 metadata 键），并镜像到 raw_mcap asset 的 `files` map、发 `asset_updated` 事件（ES 重建），使资产页/下游可见转码产物。幂等；mcap 不存在 → 404。

## Impact
- **Affected code**:
  - `backend/internal/handlers/mcap/handler.go`：新增 `PatchDerivedURIs`（参照现有 `BackfillGraceVideoID` 事务模式：Get mcap → merge metadata → Set → 镜像 asset.files → asset_updated 事件）。
  - `backend/routes/routes.go`：注册 `PATCH /api/v1/internal/mcap-files/:id/derived-uris`（admin-gated，同 grace-video-id 那组）。
  - `backend/internal/postgres/repos.go`：mcap `Set` 已写 metadata（JSONB），复用；asset `files` merge 复用 asset PATCH 的既有逻辑或 AssetRepo.Set。
  - 契约：`api/openapi.yaml` + `docs/review/api-guide.md` + smoke。
  - 只读回填脚本（不入仓库运行时）：按 md5 查 Grace → 调端点。
- **New APIs**: `PATCH /api/v1/internal/mcap-files/{id}/derived-uris`。
- **Dependencies**: 无；**无 migration**（`metadata` 是既有 JSONB 列）。

## Scope
- **In scope**: 内部 patch 端点（merge metadata.derived_uris + 镜像 asset.files + asset_updated 事件）、契约同步、单测、回填脚本（对已导入 ~10,100 条）。
- **Out of scope**: 生成 segment / ingest 切分；prod 写入；给 mcap 建「通用」全字段 update（本次只开 derived-uris 这一窄口）。

## Success Criteria
- [ ] `PATCH /internal/mcap-files/{id}/derived-uris` 传 mp4/mcap url → mcap `metadata.derived_uris` 出现这些 key，其它 metadata 键不丢。
- [ ] 镜像 raw_mcap asset 的 `files` map 含对应 url（资产页「文件」tab 可见）；asset_updated 事件已发（ES 可搜/一致）。
- [ ] 幂等（重复 patch 结果稳定）；mcap 不存在 → 404；空 body → 400。
- [ ] 回填脚本对已导入 ~10,100 条补上 forward_stereo/mezzanine/watermark（存在者）。
- [ ] openapi/api-guide/smoke 同步；Tier M/L + dev 验证。
