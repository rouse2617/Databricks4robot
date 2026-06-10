# CYB-1053~1056,1060 — Algo Runs API 扩展

## What to build

扩展 algo_runs API：list with filters、cancel、affected-assets、schema 扩展（external_runtime/external_url）、PG→ES 同步。

## Acceptance criteria

- [ ] GET /api/v1/algo-runs 支持 algo_name/status/时间范围分页
- [ ] POST /api/v1/algo-runs/{run_id}/cancel 非终态 → cancelled
- [ ] GET /api/v1/algo-runs/{run_id}/affected-assets 反查资产
- [ ] algo_runs 添加 external_runtime/external_url 列
- [ ] PG→ES projector 同步 algo_runs 数据
- [ ] OpenAPI + api-guide + smoke
- [ ] 单元测试

## Blocked by

None — CYB-1018 ingest 已完成
