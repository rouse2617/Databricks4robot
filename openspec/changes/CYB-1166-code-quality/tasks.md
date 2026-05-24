# Tasks — CYB-1166

## Context files
- `backend/internal/deliveryrules/asset_validator.go` — 精简 ParentInfo + 拒绝未知 asset_type
- `backend/internal/deliveryrules/asset_validator_test.go` — 补 derived_asset 测试
- `backend/internal/openlineage/emitter.go` — 移除不可达代码
- `backend/internal/outbox/delivery_eligibility_projector.go` — customerID 查询失败加日志

## Implementation
- [x] [validator] ValidateCreate default 分支返回 HierarchyViolation
- [x] [validator] 确认 ParentInfo.AssetType 为唯一使用字段后，移除其他字段
- [x] [validator] 更新 AssetRepoParentGetter 只读 AssetType
- [x] [validator] 更新 mockParentGetter / 测试只传需要的字段
- [x] [emitter] 移除 Emit() 中的 nil client fallback 和重复 TrimSpace
- [x] [projector] customerID 查询 err != nil 时加 slog.Warn
- [x] [test] 补 TestValidateCreate_derivedAssetParentNotFound

## API contract sync
无 HTTP API 变更

## Local verification
- [x] `cd backend && go build ./...`
- [x] `cd backend && go test ./internal/deliveryrules/ -v` — 25/25 pass
- [x] `cd backend && go test ./internal/openlineage/ -v` — 3/3 pass
- [x] `cd backend && go test ./internal/outbox/ -v` — 9/9 pass
- [x] `cd backend && go test ./...` — all packages pass

## Deploy verification (runtime changes: asset_validator default, projector log)
- [ ] Build + push with git SHA tag and `cloudrun-dev-latest`
- [ ] L1 smoke on dev

## PR
- [ ] PR template filled; Linear `CYB-1166` linked
