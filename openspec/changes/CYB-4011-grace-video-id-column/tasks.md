# Tasks — CYB-4011

Branch: `feat/CYB-4011-grace-video-id-column` (from `origin/dev`). Verification tier: **L** (postgres + handlers + OpenAPI contract + Frontend UI route).

Scope reminder: **只预留口子 + 数据可空**。加列 + 端到端读写显示过滤；**不做回填/增量同步/vibecap 直填**（既有行 `grace_video_id` 保持 NULL）。

## 1. Migration (off-limits `backend/migrations/`, 已获发起人授权)
- [ ] [migration] `backend/migrations/<YYYYMMDDHHMMSS>_grace_video_id_column.sql`（时间戳晚于 `20260722100000`）：
  - `ALTER TABLE mcap_files ADD COLUMN grace_video_id text;`
  - `ALTER TABLE assets ADD COLUMN grace_video_id text;`
  - `CREATE INDEX idx_assets_grace_video_id ON assets (grace_video_id) WHERE is_deleted = FALSE AND grace_video_id IS NOT NULL;`
  - **无** backfill DO block（本次不填充）。
- [ ] [migration] `cd backend && make db-migrate-hash`（重算 atlas.sum）。
- [ ] [migration] `atlas migrate validate --env migrate --dev-url "docker://postgres/17/dev?search_path=public"`。

## 2. Backend structs
- [ ] [backend] `dbschema/asset_models.go`：`McapFile` 结构体加 `GraceVideoID string gorm:"column:grace_video_id;type:text" json:"grace_video_id,omitempty"`（line ~256 旁）。
- [ ] [backend] `models/asset.go`：`Asset`（flatten 块 ~line 113）与 `McapFile`（~line 215）各加 `GraceVideoID string json:"grace_video_id,omitempty"`。

## 3. Backend repo — assets 读写路径 (`postgres/repos.go`)
- [ ] [backend] `Get()` SELECT 列表 + `scanOneAsset()`（var decl + `row.Scan` + hydration）加 `grace_video_id`，位置对齐。
- [ ] [backend] `GetAll()` SELECT 列表加 `grace_video_id`（同 scan 序）。
- [ ] [backend] `Set()` upsert：INSERT 列 + VALUES 占位符（renumber 尾部）+ ON CONFLICT + exec args（`nullableText(a.GraceVideoID)`）。
- [ ] [backend] `InsertNew()`：INSERT 列 + 占位符 renumber + exec args。
- [ ] [backend] `listWithFiltersData()`（/assets 列表 + facet 列表路径）：SELECT + var + `rows.Scan` + hydration。
- [ ] [backend] 确认 `ListByMcapFile()` 不含 flatten 列 —— 无需改（保持 parity）。

## 4. Backend repo — mcap_files 读写路径 (`postgres/repos.go`)
- [ ] [backend] `McapFileRepo.Get()`：SELECT `COALESCE(grace_video_id,'')` + var + Scan + hydration。
- [ ] [backend] `McapFileRepo.Set()`：INSERT 列 + 占位符 renumber + ON CONFLICT + exec args（`nullable(f.GraceVideoID)`）。
- [ ] [backend] `McapFileRepo.List()`：SELECT `COALESCE(grace_video_id,'')` + `rows.Scan(&f.GraceVideoID)`（直接扫，COALESCE 保非空）。

## 5. Backend — create 开口子 + 镜像 + ES + filter
- [ ] [backend] `handlers/mcap/handler.go`：`CreateFile` 请求结构体加 `GraceVideoID string json:"grace_video_id"`；`f := &models.McapFile{...}` 加 `GraceVideoID: req.GraceVideoID`；placeholder asset 加 `GraceVideoID: f.GraceVideoID`（镜像）。
- [ ] [backend] `filter/assets_fields.go`：`exactFieldSpecs` 加 `"grace_video_id": {Canonical, StorageField:"grace_video_id", IsJSONB:false}`；`mcapFilterColumns` 视需要加。**不**加 facet 映射。
- [ ] [backend] `searchindex/builder.go`：asset 顶层块 + mcap nested 块各加 `if …GraceVideoID != "" { doc/mcapObj["grace_video_id"] = … }`。
- [ ] [backend] **确认不改** `queryplan/planner.go`、`asset_facets.go`、`elasticsearch/query_ir.go` 的 facet 路径（D3）。

## 6. Backend tests (同 commit 更新 fixtures)
- [ ] [backend] `postgres/repos_test.go`：asset Get fixture（46→47 列，补一个 `(*string)(nil)`）；`TestListWithFilters_NoFilters`（补 NULL）；`TestMcapRepo` Get fixture + `mcapRowValues`（补 `""`）。
- [ ] [backend] `searchindex/builder_test.go`：flatten 顶层测试补 `GraceVideoID` + 期望值；omit-empty 测试的缺失键列表补 `"grace_video_id"`。
- [ ] [backend] `filter/parse_test.go`：CYB3715 flatten 用例补 `{"grace_video_id","grace_video_id"}`。

## 7. API contract sync (mandatory, same PR)
- [ ] [contract] `api/openapi.yaml`：`Asset`、`McapFile`、`McapCreateFileRequest` schema 各加 `grace_video_id`（type string，描述 = Grace video UUID，可空）。
- [ ] [contract] `docs/review/api-guide.md`：mcap create / get 段落补 `grace_video_id` 字段说明 + 一个过滤示例 `filter=grace_video_id:eq:<uuid>`。
- [ ] [contract] smoke：`scripts/smoke-*` 或 api-guide-smoke 加一条 —— create 带 grace_video_id → get 读回 → filter 命中。
- [ ] [contract] SDK：如公共 mcap client 暴露该字段，`sdk/src/...` + `sdk/tests/unit/` 同步；若不涉及公共面，在 decisions.md 声明。

## 8. Frontend (数据可空，显示「未关联」)
- [ ] [frontend] `Frontend/src/api/types.ts`：`McapFile`（+ `Asset` 如列表需要）加 `grace_video_id?: string`。
- [ ] [frontend] `components/mcap/McapDetailDrawer.tsx`：加 `Descriptions.Item label="Grace Video ID"` —— 有值渲染可复制文本 + 跳转 `https://grace.cyberorigin.ai/videos/<id>`，空值渲染「未关联」。
- [ ] [frontend] `components/assets/AddFilterPopover.tsx`：加 `{ key:"grace_video_id", label:"Grace Video ID", type:"string" }`（filter-only）。
- [ ] [frontend] **不**加 facet 侧栏 / `useAssetsDiscoveryReducer` facet 请求（D3）。

## 9. Verify (Tier L)
- [ ] [backend] `make fmt && make vet && go test ./...`。
- [ ] [frontend] `npm run lint && npm run build`（+ 相关 vitest）。
- [ ] [sdk] 若触及：`cd sdk && uv run pytest tests/unit/`。

## 10. Deploy verify (dev, before commit)
- [ ] `bash scripts/apply-migration-dev.sh "$(pwd)/backend/migrations/<new>.sql"` → smoke query 确认列存在。
- [ ] `bash deploy/cloudrun/backend-dev.sh` → smoke：create mcap 带 grace_video_id → get 读回 → `/queries/run` filter 命中。
- [ ] `bash deploy/cloudrun/frontend-dev.sh` → **Chrome DevTools MCP** 在 dev 打开一个 mcap 详情，确认 Grace Video ID 项显示（有值可跳转 / 空值「未关联」）。
- [ ] 等用户确认后再 commit/push + 开 PR（填 PR 模板，列出契约同步行）。
