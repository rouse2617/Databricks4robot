# Tasks — CYB-4445

## Backend

### Interface + repo
- [ ] `backend/internal/repository/common.go` — `McapFileRepository.List` 增加 `mcapFileID string` 末位参数
- [ ] `backend/internal/postgres/repos.go` — `McapFileRepo.List` 签名同步；WHERE 子句加 `AND mcap_file_id = $N` 段，沿用 arg-index counter

### Handler
- [ ] `backend/internal/handlers/mcap/handler.go` — `ListFiles` 读 `c.Query("mcap_file_id")`，非空时正则校验 `^[A-Za-z0-9]{8}$`，不匹配返回 `400 INVALID_ARGUMENT`；将值原样传给 `repo.List`
- [ ] `backend/internal/handlers/mcap/handler.go` — Swagger 注释（若 handler 用了 `@Param` 风格）追加 `mcap_file_id`

### Tests
- [ ] `backend/internal/handlers/mcap/handler_test.go` — `mockMcapRepo.List` 签名同步；新增 `TestListFiles_McapFileIDFilter`（happy exact match → 200 + 1 条；空 → 不加参；非 8 位 → 400；与 owner 组合交集）
- [ ] `backend/internal/postgres/repos_test.go` — `TestMcapFileRepo_List_ReadsRealColumns` 与 `TestMcapFileRepo_List_FiltersByRealColumns` 增加 `mcapFileID` 末位参数；新增对 `mcap_file_id = $N` 拼装的断言（mock 查询语句 + 命中场景）
- [ ] `backend/internal/searchindex/builder_test.go:97` — `stubMcapRepo.List` 签名同步（新增 `mcapFileID` 末位参数，桩内忽略）

### Changelog
- [ ] `backend/CHANGELOG.md` — `[Unreleased] → Added` 追加："MCAP files list: `mcap_file_id` exact-match filter (`GET /api/v1/mcap-files?mcap_file_id=...`) — and-combined with owner and ingest_state; 8-char alphanumeric enforced"

## Frontend

- [ ] `Frontend/src/api/mcapFiles.ts` — `ListMcapFilesParams` 增加 `mcap_file_id?: string`，`list()` 实现按需 append 到 URL
- [ ] `Frontend/src/pages/McapFilesPage.tsx`
  - [ ] 加 `mcapFileIdFilter` + `debouncedMcapFileIdFilter` 状态（仿 `ownerFilter`）
  - [ ] 客户端正则校验；不足 8 位强制 `debouncedMcapFileIdFilter = ""`（不下推）
  - [ ] 新增 `Input`（`maxLength={8}`，placeholder「按文件 ID 搜索（8 位）」，hint 「请输入 8 位字母数字 ID」 当长度>0 且 ≠8）
  - [ ] 加到现有过滤行（栅格列宽按场景自适应）
  - [ ] `debouncedMcapFileIdFilter` 变化 → `setPage(1)`（同 owner 模式）
  - [ ] `mcapFilesApi.list(...)` 调用里 `mcap_file_id: debouncedMcapFileIdFilter || undefined`
- [ ] 不要碰 `?mcap_file_id=...` URL 直跳详情抽屉的已有逻辑

## API contract sync (mandatory, same PR)

- [ ] `api/openapi.yaml` — `GET /mcap-files` parameters 加 `mcap_file_id`（in: query, type: string, pattern: `^[A-Za-z0-9]{8}$`, description, example `LEMpjOmB`）；400 响应示例
- [ ] `docs/review/api-guide.md` — 5.2 MCAP 列表节追加：
  - [ ] curl：`GET /api/v1/mcap-files?mcap_file_id=LEMpjOmB`
  - [ ] curl：`GET /api/v1/mcap-files?mcap_file_id=SHORT` → 400
  - [ ] curl：与 owner 组合交集
  - [ ] 字段说明：精确 8 位、AND 组合、`total` 反映交集
- [ ] `Frontend/src/api/types.ts` — `McapFile` / `PaginatedResponse<T>` 形状不需变；确认在 PR body 中声明 "no type changes; aligned via existing types"

## Verification

### Static / unit (Tier L — API 同步 + 多文件)
- [ ] `cd backend && make fmt && make vet`
- [ ] `cd backend && go test ./internal/handlers/mcap/... ./internal/postgres/... ./internal/repository/... ./internal/searchindex/...`
- [ ] `cd Frontend && npm run lint`
- [ ] `cd Frontend && npm run test -- --run -- McapFilesPage mcapFiles`
- [ ] `cd Frontend && npm run build`

### Smoke (dev)
- [ ] `source scripts/dev-backend-env.sh`
- [ ] `curl "$API_BASE/api/v1/mcap-files?mcap_file_id=LEMpjOmB" -H "X-Databrew-Token: $TOKEN" | jq '.total'`
- [ ] `curl -o /dev/null -w '%{http_code}\n' "$API_BASE/api/v1/mcap-files?mcap_file_id=SHORT" -H "X-Databrew-Token: $TOKEN"` → 400

### UI (per `docs/agents/deploy-before-commit.md` + Chrome DevTools MCP mandatory for Frontend diff)
- [ ] Deploy backend dev: `bash deploy/cloudrun/backend-dev.sh` + `gcloud run deploy … --service-account=…`（按 `current-work.md` §"Cloud Run Deployment"）
- [ ] Deploy frontend dev: `wrangler deploy --env dev`
- [ ] Chrome DevTools MCP:
  - [ ] open `https://cyber-databrew.cyberorigin.ai/mcap-files`
  - [ ] type `LEMpjOmB` into new input → assert 1 row, total = 1
  - [ ] type `LEMpjO` (7 chars) → assert table returns to unfiltered list
  - [ ] type `SHORT` → assert API 400 (curl-equivalent) plus UI behavior unchanged
  - [ ] paste the screenshot path into the PR description

## OpenSpec close-out

- [ ] PR 合并后，把 `openspec/changes/CYB-4445-mcap-files-id-search/specs/mcap-files/spec.md` ADDED/MODIFIED 段并入 `openspec/specs/mcap-files/spec.md`（如果不存在该基线 spec，则先创建简化版；按 spec-driven-workflow.md「Archive」）

## Decisions 记录

- [ ] `decisions.md` 追加：精确匹配 vs 模糊匹配 的选择（参见 `design.md` 末表）
- [ ] `decisions.md` 追加：保留 `/mcap-files/:id` 路由缺失为独立 issue，不在本次合并
