# Proposal — CYB-1166

## Why

代码 review 中发现 4 个模块有可清理的小问题：不可达代码、误导性数据结构、缺失校验、错误被静默吞掉。

## What Changes

### Modified Capabilities
- **asset_validator.go**: ValidateCreate 的 default 分支改为拒绝未知 asset_type（之前静默通过）
- **asset_validator.go**: 精简 ParentInfo，移除未被使用的字段
- **emitter.go**: 移除不可达 nil client fallback 和重复 TrimSpace
- **delivery_eligibility_projector.go**: customerID 查询失败时加 slog.Warn 日志

### New Tests
- derived_asset parent-not-found 测试用例

## Impact
- asset_validator.go 的行为变化：未知 asset_type 的 Create 请求之前会通过校验，现在返回 HierarchyViolation
- 其他变更均为纯删除/加日志，行为兼容

## Scope
- **In scope**: 4 个文件的小清理
- **Out of scope**: 架构改动、新抽象、N+1 优化（P1.5 TODO 保留）

## Success Criteria
- [ ] `go build ./...` passes
- [ ] `go test ./internal/deliveryrules/` passes
- [ ] `go test ./internal/openlineage/` passes
- [ ] `go test ./internal/outbox/` passes
