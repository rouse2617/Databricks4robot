# Proposal — CYB-4445: MCAP 文件列表按 ID 搜索

## Why

`/mcap-files` 页面目前只支持按 `owner`（模糊包含）和 `ingest_state`（精确） 过滤。21k+ 条记录下，已知 8 位 MCAP 文件 ID 的用户无法在 UI 上直接定位，必须手工构造 `?mcap_file_id=<id>` URL 或挨页翻。用户原话：「ID 检索入口，你来开发个」。

## What Changes

### New Capabilities
- **mcap-files-id-search**: 用户可以在 `/mcap-files` 页面顶部新增的「按文件 ID 搜索」输入框直接通过 8 位 MCAP ID 定位某一条记录；同时保持现有 owner / 状态过滤与原有 URL 直跳详情抽屉行为不变。

### Modified Capabilities
- **(无)** 现有 `mcap-files` 列表的响应结构与 owner/state 过滤行为完全不变；本次只新增一个可选查询参数。

## Impact
- **Affected code**:
  - `backend/internal/handlers/mcap/handler.go` — `ListFiles` 处理 `mcap_file_id` query
  - `backend/internal/repository/common.go` — `McapFileRepository.List` 接口签名加一个参数
  - `backend/internal/postgres/repos.go` — `McapFileRepo.List` 实现 + WHERE 子句
  - `backend/internal/postgres/repos_test.go` — repo 测试
  - `backend/internal/handlers/mcap/handler_test.go` — handler 测试（含 `mockMcapRepo` 桩签名更新）
  - `backend/internal/searchindex/builder_test.go` — `stubMcapRepo` 桩签名更新（接口签名变更连带）
  - `Frontend/src/api/mcapFiles.ts` — `ListMcapFilesParams` 增加 `mcap_file_id`
  - `Frontend/src/pages/McapFilesPage.tsx` — 第三个过滤输入框 + debounce + 校验
  - `api/openapi.yaml` — `GET /mcap-files` parameters 新增 `mcap_file_id`
  - `docs/review/api-guide.md` — 5.2 节加 curl 与 400 错误示例
  - `backend/CHANGELOG.md` — `[Unreleased] → Added` 追加一条

- **New APIs**:
  - `GET /api/v1/mcap-files?mcap_file_id=<8位字母数字>`（已有 endpoint 新增可选 query 参数；非新路由）

- **Dependencies**: 无（前端不引入新依赖；后端沿用现有 `pgx`/`Gin`）

## Scope
- **In scope**:
  - 后端 `GET /api/v1/mcap-files` 支持 `mcap_file_id` 精确匹配（`=`）；与 `owner`、`ingest_state` AND 组合
  - 服务端对 `mcap_file_id` 做与 schema CHECK 一致的 8 位字母数字校验；非空非匹配返回 `400 INVALID_ARGUMENT`
  - 前端页面增加 8 位限制输入框 + 客户端同源校验 + debounce
  - OpenAPI 与 api-guide 在同一 PR 同步

- **Out of scope**:
  - 修复 `Frontend/src/components/CmdKSearch.tsx:94–99` 把 `MCAP <id>` 跳到 `/mcap-files/${id}`（该路由 `App.tsx:61` 未注册，会落到 `*` 重定向到 `/dashboard`）。这是独立 bug，记为后续 issue，不在本次范围内处理。
  - 把 `owner` 从模糊匹配改成精确匹配。
  - 给其他列表页（`asset_files`、`algo-runs` 等）加 ID 搜索。
  - 服务端用 `ILIKE` 提供 ID 子串模糊匹配（短字符串子串容易误匹配，与现有 `CmdKSearch.isAssetId` 语义也不一致）。

## Success Criteria
- [ ] 用户在 `/mcap-files` 输入 `LEMpjOmB`（或任何已存在的 8 位 ID）后，表格收敛到 1 条，`total === 1`。
- [ ] 输入 `LEMpjO`（7 位）不会出现「无结果」误导，列表保持原样（参数不下推）。
- [ ] 同时输入 `mcap_file_id=LEMpjOmB` + `owner=grace-pu&ingest_state=summarized` 时，结果是两个过滤条件 AND 后的交集。
- [ ] 非 8 位非空（例如 `mcap_file_id=SHORT`）直接打 `GET /api/v1/mcap-files` 返回 `400 INVALID_ARGUMENT`，错误体里给出原因。
- [ ] 现有 `?mcap_file_id=...` 直跳详情抽屉的行为保持完全不变。
- [ ] `CmdK ⌘K` 全局搜索依然能为 MCAP ID 提供「直接匹配」入口（即使落地 404，也是单独 P2，不在本次验收内）。

## Goals (SLO) — 列表场景
- **Latency**: 新增 1 个等值过滤条件，索引在前（`mcap_files.mcap_file_id` 是主键，恒定时间），新增参数后 p95 不退化。
- **Concurrency**: 无新增连接 / 信号量；现有列表查询路径不变。
- **Quality**: 单元测试覆盖 — 精确命中、空、非 8 位 400、与 owner/state 组合交集；Chrome DevTools MCP 实测 dev 一次。
