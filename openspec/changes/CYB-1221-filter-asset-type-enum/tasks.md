# Tasks — CYB-1221

## Context files
- `backend/internal/filter/types.go` — asset_type filter 枚举定义
- `backend/internal/deliveryrules/asset_validator.go` — 合法 asset_type 值的参考实现

## Implementation

- [ ] `[backend]` 修正 `types.go:47` `SearchableFields["asset_type"].Values`：补全 `raw_mcap`、`action`、`task`，`frame_set` → `frame`

## Verification

- [ ] `[backend]` Tier S: `cd backend && make fmt && make vet`
- [ ] `[backend]` Tier M: `cd backend && go test ./internal/filter/...`
- [ ] `[backend]` Tier L: `cd backend && go build ./...`

## Deploy verification
- [ ] 部署 backend dev，curl 验证 search API 接受全部 7 个 asset_type 值
