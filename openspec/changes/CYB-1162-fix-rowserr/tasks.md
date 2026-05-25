# Tasks — CYB-1162

## Context files
- `backend/internal/handlers/asset/lineage_response.go` — 目标文件：三处 rows.Next() 循环
- `backend/internal/handlers/asset/handler_test.go` — 已有 fakeAssetSQLRows / fakeAssetSQLQuerier 测试辅助工具

## Implementation
- [x] [backend] 在 lineage_response.go 三处 `for rows.Next()` 循环后添加 `rows.Err()` 检查
- [x] [backend] 用 `slog.Warn` 记录扫描错误而非静默跳过（line 71, 91, 112 的 `err == nil` 守卫）
- [x] [backend] 新增 `TestBuildLineageResponse` 单元测试覆盖：
  - [x] Happy path: 三个查询都返回有效行
  - [x] rows.Err 场景: algo_results 查询在迭代后返回 err，验证日志记录
  - [x] pg == nil 场景: 无数据库连接时返回空结构

## API contract sync
本 change 不涉及 HTTP API 变更 — 无需 contract sync。

## Local verification (Tier M per AI-RULES)
- [x] `cd backend && go build ./...`
- [x] `cd backend && go test ./internal/handlers/asset/ -run TestBuildLineageResponse -v`

## Deploy verification (before commit — runtime only)
- [x] Build + push with git SHA tag and `cloudrun-dev-latest` per `docs/agents/deploy-before-commit.md`

### Backend (if `backend/` changed)
- [ ] L1 smoke (`healthz`, core APIs) on dev

## PR
- [ ] PR template filled; Linear `CYB-1162` linked
