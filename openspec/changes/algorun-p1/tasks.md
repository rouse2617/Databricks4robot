# Tasks — AlgoRun MVP Phase 1

## Context files
See `context-files.md`

## Implementation

### Phase 0 完成后
- [ ] 确认 migration 044 已合并到 feat/pipeline-integration（relation_type + metadata + ml_model schemas）

### Migration 045
- [ ] [backend] 新增 `algo_run_inputs` 表（run_id, input_asset_id, role, created_at）
- [ ] [backend] 新增 `algo_run_outputs` 表（run_id, artifact_name, asset_id, logical_asset_id, artifact_type, metadata, created_at）
- [ ] [backend] 幂等键索引 (run_id, artifact_name)

### Repository 层
- [ ] [backend] `algorun_artifact` 新增 repository interface（InsertOutput, ListOutputs, GetOutputByArtifactName）
- [ ] [backend] PG 实现：InsertOutput（事务内写 algo_run_outputs + assets + asset_relations + asset_events）

### Usecase 层
- [ ] [backend] `RegisterOutput` usecase — 核心事务：
  - 校验 run 状态为 ok
  - 创建/复用 logical_asset（可按 artifact_name 匹配）
  - INSERT assets row（ml_model/dataset 类型）
  - revision += 1
  - INSERT asset_relations（input→output 血缘）
  - INSERT algo_run_outputs
- [ ] [backend] 幂等处理：同 (run_id, artifact_name) 返回既有资产，不重复创建
- [ ] [backend] `ListOutputs` — 查询 run 的所有产物

### Handler 层
- [ ] [backend] `RegisterOutputHandler` — POST /api/v1/algo-runs/{run_id}/outputs:register
- [ ] [backend] `ListOutputsHandler` — GET /api/v1/algo-runs/{run_id}/outputs

### Routes
- [ ] [backend] routes.go 注册 2 个新路由

## API contract sync
- [ ] `api/openapi.yaml` — paths + schemas（RegisterOutputRequest/Response, AlgoRunOutput）
- [ ] `docs/review/api-guide.md` — curl 示例 + 错误码
- [ ] `scripts/api-guide-smoke.sh` — 烟雾测试新增项

## Local verification
- [ ] `make fmt && make vet` in backend/
- [ ] `go test ./...` 全部通过
- [ ] 单元测试：RegisterOutput happy path + 幂等 + 非法 run 状态 + 不存在的输入资产

## Deploy verification
- [ ] 构建镜像 + push
- [ ] 部署到 Cloud Run dev
- [ ] 烟雾测试通过（34/35 + 新增的 AlgoRun 项）
