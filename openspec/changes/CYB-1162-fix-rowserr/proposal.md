# Proposal — CYB-1162

## Why

`buildLineageResponse` 在遍历 SQL 结果集后不检查 `rows.Err()`，若迭代过程中游标提前失败，部分下游数据（algo_results / deliveries / eval_results）会被静默截断而不记录任何错误。这与 commit `7f0cf5e` 在审计搜索中修复的 bug 类别相同。

## What Changes

### New Capabilities
- None.

### Modified Capabilities
- **asset-management**: `buildLineageResponse` 在遍历 SQL 行后检查 `rows.Err()`，并在扫描错误时以日志形式记录而非静默跳过。

## Impact
- **Affected code**: `backend/internal/handlers/asset/lineage_response.go`
- **New APIs**: None; 无 HTTP 行为变更
- **Dependencies**: None

## Scope
- **In scope**: lineage_response.go 中三处 rows.Next() 循环后添加 rows.Err() 检查；添加扫描错误日志；新增单元测试覆盖 buildLineageResponse
- **Out of scope**: 任何 HTTP API 行为变更；其他 handler 的 bug 修复

## Success Criteria
- [ ] `buildLineageResponse` 在迭代后检查 `rows.Err()`，游标错误不再静默丢失
- [ ] 扫描错误至少以日志形式记录而不被忽略
- [ ] 新增 `TestBuildLineageResponse` 覆盖 happy path + rows.Err 场景
