# Tasks — CYB-1222

## Context files
- `backend/routes/routes.go` — 路由注册
- `backend/internal/handlers/asset/handler.go` — handler 实现（参考 Create + actionHandler.Create）
- `backend/internal/usecase/asset/usecase.go` — usecase 层（参考现有 Create 实现）
- `backend/internal/deliveryrules/asset_validator.go` — L1-L7 校验
- `docs/review/unified-asset-catalog/design/asset-hierarchy-and-derivatives.md` §3.2.1, §4.2

## Implementation

### Validator 层
- [x] `[backend]` 添加 L5 checkTask 允许 task→task（asset_validator.go）

### Repository 层
- [x] `[backend]` 添加 InsertRelation 到 AssetRelationWriter + AssetRepo 实现

### Handler 层
- [x] `[backend]` 添加 `CreateClip` handler — POST /assets/:id/clips
- [x] `[backend]` 添加 `CreateAction` handler — POST /assets/:id/actions（代码就绪，路由暂未注册——被 actionH 占用）
- [x] `[backend]` 添加 `CreateFrame` handler — POST /assets/:id/frames
- [x] `[backend]` 添加 `CreateTask` handler — POST /assets/:id/tasks

### Usecase 层
- [x] `[backend]` 添加 `CreateChildAsset` usecase — 同事务创建 asset + asset_relations + events
- [x] `[backend]` 实现 split_method 决策表（algo:* → derived_from, 其他 → split_from）
- [x] `[backend]` 连接 AssetWriteValidator 做 L1-L7 校验

### Routes
- [x] `[backend]` routes.go 注册 3 个新路由（/clips, /frames, /tasks），/actions 因路由冲突暂未注册

## API Contract Sync (mandatory for new endpoints)
- [x] `api/openapi.yaml` — 添加 3 个新 path + ChildAssetCreateRequest schema
- [x] `docs/review/api-guide.md` — 添加 §1.10 分层 API 章节 with curl 示例
- [x] `scripts/api-guide-smoke.sh` — 添加 3 个端点 happy path + error path（404）
- [ ] `openspec/changes/CYB-1222-layered-api/specs/api/spec.md` — 已完成

## Verification
- [x] `[backend]` Tier S: `make fmt && make vet`
- [x] `[backend]` Tier M: `go test ./internal/deliveryrules/...` + `go test ./internal/handlers/asset/...`
- [x] `[backend]` Tier L: `go build ./...` + `go test ./...`

## Deploy verification
- [x] 部署 backend dev
  - Image: `cyber-databrew-backend:bbdd90b@sha256:5b823a642fcaa5a72b6c85225835cdafb931a0bc47aabf65e9d9af77039ee12f`
  - Revision: `cyber-databrew-backend-dev-00221-5pk`
  - URL: `https://cyber-databrew-backend-dev-234851712830.us-central1.run.app`
- [x] 测试 3 个端点的 happy path（segment 下创建 clip/frame/task → 201）
- [x] 测试 L5 task→task 层级（task 下创建 task → 201）
- [x] 测试 error path（不存在父 → 404，无效时间戳 → 422）
- [x] 运行回归 smoke（api-guide-smoke.sh 31 passed, 1 failed — healthz is pre-existing）
