# Tasks — CYB-4271

Branch: `feat/CYB-4271-mcap-derived-uris` (from `origin/dev`). Verification tier: **M** (postgres metadata/files round-trip + handler + OpenAPI contract; no migration, no frontend route change).

## Context files
- `backend/internal/handlers/mcap/handler.go` — `BackfillGraceVideoID` (line 467) is the template tx/mirror/event pattern; `PatchDerivedURIs` added after it.
- `backend/internal/postgres/repos.go` — `McapFileRepo.Get/Set` (metadata round-trip), `AssetRepo.Set`/`bindAssetJSONAndRefs` (line 198, FilesJSON is the marshal source), `SyncLegacyFields`.
- `backend/routes/routes.go` — `internalMcap` group (line 394), admin-gated.
- `backend/internal/handlers/mcap/handler_test.go` — `TestBackfillGraceVideoID` (line 465) fakes reused by `TestPatchDerivedURIs`.
- `api/openapi.yaml` — grace-video-id path (~5801) + McapCreateFileRequest schema (~1366) as templates.
- `docs/review/api-guide.md` §7.2.2b (grace-video-id) — §7.2.2c added below it.
- `scripts/api-guide-smoke.sh` — RUN_WRITES block (~616); `patch_json`/`expect_code_patch` helpers added.

Scope reminder: **只开 derived-uris 这一窄口**。merge `metadata.derived_uris` + 镜像 asset `files` + 发 `asset_updated`；**不做** segment 切分 / prod 写入 / 通用全字段 mcap update。**无 migration**（`metadata`/`files_json` 是既有 JSONB）。

派生 URL key（body 与 `metadata.derived_uris`/`files` 内统一命名）：
`raw_mcap_uri`、`forward_stereo_mp4`、`mezzanine_mp4`、`watermark_mp4`、`imu_mcap_uri`（全部可选，至少传一个）。

## 1. Backend — handler (`backend/internal/handlers/mcap/handler.go`)
- [ ] [backend] 新增 `PatchDerivedURIs(c *gin.Context)`，参照 `BackfillGraceVideoID`（line 467）事务模式：
  - 解析 `:id`；`ShouldBindJSON` 到请求结构体（上述 5 个 `json:"...,omitempty"` string 字段）；`TrimSpace` 各值；收集非空成一个 `map[string]string`；空集合 → `BadRequest`(CodeInvalidArgument)。
  - `h.repo.Get(ctx, id)`；nil → `NotFound("MCAP_FILE_NOT_FOUND")`。
  - `withTx`：
    1. merge 进 `f.Metadata["derived_uris"]`（若 `f.Metadata` 为 nil 先建 map；`derived_uris` 子 map 逐 key 覆盖，其它 metadata 键不动）→ `h.repo.Set(txCtx, f)`。
    2. `assetRepo.Get(id)`：存在且非删除 → merge 相同 key 进 `a.Files`（nil 则建）→ `assetRepo.Set`。
    3. merge 后有实际变更时 `eventRepo.Append(asset_updated)`（payload 带 `asset_id` + `derived_uris`），复用 BackfillGraceVideoID 的 append 形态。
  - 幂等：merge 前后完全一致 → 直接 200 返回当前 mcap（不发事件）。
- [ ] [backend] doc-comment 说明窄口用途 + 路由（照 BackfillGraceVideoID 注释风格）。

## 2. Backend — route (`backend/routes/routes.go`)
- [ ] [backend] 在 grace-video-id 同组（`internalMcap`，`adminRoutesEnabled` gated，line ~396）注册 `internalMcap.PATCH("/mcap-files/:id/derived-uris", mcapHandler.PatchDerivedURIs)`。

## 3. Backend — repo 复用确认（不改 schema）
- [ ] [backend] 确认 `McapFileRepo.Get/Set` 已完整 round-trip `metadata`（JSONB）—— merge 后 Set 不丢其它 metadata 键（Set 是 upsert，回写 Get 读到的全字段）。
- [ ] [backend] 确认 `AssetRepo.Get/Set` 已完整 round-trip `files`/`files_json`（JSONB）—— 若 `Set` 不写 `files_json`，改走 asset PATCH 既有 files-merge 逻辑（对齐 `PATCH /assets/:id` 的 files 更新路径），避免只更 metadata 不落 files。**先读代码确认，再决定复用哪条路径。**

## 4. Backend tests（同 commit）
- [ ] [backend] `handlers/mcap/*_test.go`：新增 `PatchDerivedURIs` 用例 —— merge 保留无关 metadata 键 / 空 body → 400 / 不存在 → 404 / 幂等重复 patch。
- [ ] [backend] 若 repo round-trip 有疑点，`postgres/repos_test.go` 补一条 mcap metadata.derived_uris + asset files 的 upsert-回读断言。
- [ ] [backend] `searchindex/builder_test.go`：若 `files`/`derived_uris` 进 ES 文档，补断言；若不进 doc，则在 decisions.md 声明「派生 URL 不进 ES 检索面，仅落库+资产页」。

## 5. API contract sync (mandatory, same PR)
- [ ] [contract] `api/openapi.yaml`：新增 `McapDerivedURIsRequest`（5 个可选 string，描述各 URL 来源）；path `PATCH /internal/mcap-files/{id}/derived-uris`（200 返回 `McapFile`，400/404）。若 `McapFile.metadata` 未在契约展开 `derived_uris`，补一条说明。
- [ ] [contract] `docs/review/api-guide.md`：internal mcap 段落补该端点 —— 请求示例 + 「merge 语义/幂等/仅 admin」说明。
- [ ] [contract] smoke（`scripts/smoke-*` 或 api-guide-smoke）：create mcap → PATCH derived-uris → get 读回 `metadata.derived_uris` 命中；重复 PATCH 幂等。
- [ ] [contract] SDK：本端点为 internal/admin，若公共 SDK 不暴露 internal 面，则在 decisions.md 声明不同步 SDK。

## 6. Backfill script（只读 Grace + 调端点，不入仓库运行时）
- [ ] [ops] 脚本（一次性，放 `scripts/` 或本地）：对 `owner=grace-pull-demo` 的已导入 mcap，按 `raw_hash_md5`(=md5) 反查 Grace prod `storage_meta`，取：
  - `storage_meta.gcs.video` → `raw_mcap_uri`
  - `storage_meta.gcs.algo_inputs.mcap.uri` → `forward_stereo_mp4`
  - `storage_meta.gcs.annot_inputs.right.uri` → `mezzanine_mp4`
  - aliyun oss watermark → `watermark_mp4`
  - `storage_meta.gcs.imu` → `imu_mcap_uri`（存在者）
  - 调 `PATCH /internal/mcap-files/{id}/derived-uris`；限速、幂等重跑安全、统计成功/跳过/失败。
- [ ] [ops] **Grace prod key 不回显、不入文件、不入仓库**；只读 Grace，只写 dev DataBrew。

## 7. Verify (Tier M)
- [ ] [backend] `CGO_ENABLED=0 make fmt && make vet && CGO_ENABLED=0 go test ./...`。
- [ ] [contract] api-guide-smoke（若在 CI 内联，确认新 case pass）。

## 8. Deploy verify (dev, before commit)
- [ ] `bash deploy/cloudrun/backend-dev.sh` → 用一条已导入 mcap 实测：PATCH derived-uris → get 读回 `metadata.derived_uris` + 资产 `files` 命中 → 幂等重跑。
- [ ] **Chrome DevTools MCP** 在 dev 打开该 mcap/资产详情，确认转码 mp4 / 派生 URL 可见。
- [ ] 等用户确认后再 commit/push + 开 PR（填 PR 模板，列出契约同步行）。回填脚本在 PR 合并、端点上线后单独跑。
