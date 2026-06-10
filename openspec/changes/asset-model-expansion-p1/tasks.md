# Tasks — 资产模型扩展 Phase 1

## Implementation
- [ ] Extend `asset_relations.relation_type` CHECK 约束，增加 `annotated_from`, `materialized_from`
- [ ] 引入 code-based `asset_type_schemas` registry（Go package）
- [ ] 注册 `dataset` asset_type 及字段：format, record_count, size_bytes, annotation_status, time_range, source
- [ ] 注册 `annotation_result` asset_type 及字段：tool, schema_version, annotators, quality_score, coverage
- [ ] 写入校验从硬编码 switch 改为 registry-driven validation
- [ ] ES mapping 增加 `dataset.*`, `annotation_result.*` projection
- [ ] 新增 `GET /api/v1/asset-types/{type}/schema` 端点

## API Contract Sync
- [ ] `api/openapi.yaml` — 新增 asset-types schema 端点
- [ ] `docs/review/api-guide.md` — curl 示例

## Verification
- [ ] `make fmt && make vet`
- [ ] `go test ./...` (touched packages)
- [ ] `npm run build` (if frontend touched)

## Deploy
- [ ] Deploy backend dev + smoke test
