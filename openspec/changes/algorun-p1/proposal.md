# Proposal — AlgoRun MVP Phase 1

## Why
训练系统产出（ml_model/dataset）无法自动注册到资产平台，缺失训练输入→产出的血缘闭环，导致模型版本管理和依赖追溯断裂。

## What Changes

### New Capabilities
- `algo_run_inputs` / `algo_run_outputs` 表，记录训练的输入角色和输出资产
- `POST /api/v1/algo-runs/{run_id}/outputs:register` — 训练产出注册为 ml_model/dataset 资产
- `GET /api/v1/algo-runs/{run_id}/outputs` — 查询 run 产物列表
- 自动写 input → output 资产血缘边（trained_from / evaluated_on / configured_by）

### Modified Capabilities
- `asset_relations.relation_type` 新增 trained_from/evaluated_on 等（Phase 0 已完成）
- 资产版本逻辑复用（logical_assets revision +1）

## Impact
- **Affected code**: `backend/internal/handlers/algorun/`, `backend/internal/usecase/asset/`
- **New APIs**: `POST /api/v1/algo-runs/{run_id}/outputs:register`, `GET /api/v1/algo-runs/{run_id}/outputs`
- **New tables**: `algo_run_inputs`, `algo_run_outputs` (migration 045)
- **Dependencies**: 依赖 Phase 0 的 relation_type 扩展 + ml_model/dataset 类型 schema

## Scope
- **In scope**: 训练系统通过 API 接入资产闭环，不依赖 Pipeline UI
- **Out of scope**: Pipeline 训练节点、run 对比、模型发布治理、ES 投影

## Success Criteria
- [ ] 调用 `outputs:register` 后创建新 assets row + logical_assets revision +1
- [ ] asset_relations 包含 trained_from / evaluated_on 关系边
- [ ] GET /algo-runs/{run_id}/outputs 返回输出资产列表
- [ ] 幂等：同 (run_id, artifact_name) 重复调用不创建重复资产
- [ ] 烟雾测试通过
