# Context — CYB-4445

> 实施前先读这些路径，建立共识后再动键盘。

## Repository 结构
- 后端布局：`backend/internal/{handlers,repository,postgres,...}`
- 前端布局：`Frontend/src/{api,pages,components,...}`
- API 合约：`api/openapi.yaml`、`docs/review/api-guide.md`

## 已读关键文件（实施阶段不需要再通读，直接复用）

| 文件 | 角色 |
|------|------|
| `backend/internal/handlers/mcap/handler.go:411 ListFiles` | 现有只接 `ingest_state`、`owner`；本次加 `mcap_file_id` 同位置校验 + 下传 |
| `backend/internal/handlers/mcap/handler.go:260` | 同类 8 位校验 + `CodeInvalidArgument` 用法 |
| `backend/internal/postgres/repos.go:1147 McapFileRepo.List` | 现有动态 WHERE + arg-index counter 模式 |
| `backend/internal/repository/common.go:33` | 接口签名要改的地方 |
| `backend/internal/postgres/repos_test.go:2090-2199` | repo 单测模板，照着加 |
| `backend/internal/handlers/mcap/handler_test.go:21,45,395` | handler 单测 + `mockMcapRepo` 桩 |
| `backend/internal/searchindex/builder_test.go:86,97` | `stubMcapRepo` 桩 |
| `Frontend/src/pages/McapFilesPage.tsx:73-113` | 现 owner filter 状态机 + debounce + page=1 重置 |
| `Frontend/src/pages/McapFilesPage.tsx:286-326` | 现 filter 行栅格；本次扩展为 4 列 |
| `Frontend/src/api/mcapFiles.ts:4-21` | `ListMcapFilesParams` 与 `list()` 现实现 |
| `Frontend/src/components/CmdKSearch.tsx:34-37` | 既有 `^[A-Za-z0-9]{8}$` 正则，本次复用同源 |
| `api/openapi.yaml:4880-4946` | `GET /mcap-files` parameters + response；新增 query 参数；400 响应示例 |
| `docs/review/api-guide.md:1718-1749` | 5.2 节 MCAP 列表；本次追加示例和说明 |

## 跳过（不在本次范围）

- `frontend/src/components/CmdKSearch.tsx:94-99` — `/mcap-files/${q}` 路由 404 问题，单独立 issue
- `frontend/src/App.tsx:61` 路由表 — 不动
- `openspec/specs/` 基线 spec — 暂不动，留到 PR 合并后的 archive 步骤

## 关联
- Linear：[CYB-4445](https://linear.app/cyberorigin/issue/CYB-4445/mcap-文件列表增加按-id-搜索入口)
- Git branch：`feat/CYB-4445-mcap-files-id-search`（基于 `origin/dev` 3117c85b）
