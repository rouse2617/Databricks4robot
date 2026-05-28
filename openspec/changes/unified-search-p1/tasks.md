# Tasks — 统一检索 P1：血缘 ES 投影

## Implementation
- [x] ES mapping 扩展：asset 文档增加 `lineage_upstream_ids`, `lineage_downstream_ids`, `lineage_relation_types`
- [x] asset_relations 变更时异步重建关联资产文档的 lineage 投影（通过 outbox/event）
- [x] 搜索 API 增加 lineage filter 参数：`lineage_with`, `lineage_direction`, `lineage_depth`
- [x] Search handler/usecase 扩展 upstream/downstream asset 检索
- [x] 保持 `/audit/lineage-search` 端点兼容

## API Contract Sync
- [x] `api/openapi.yaml` — 搜索 API 扩展 lineage 参数
- [x] `docs/review/api-guide.md` — curl 示例
- [x] `scripts/api-guide-smoke.sh` — lineage 搜索验证

## Verification
- [ ] `make fmt && make vet`
- [ ] `go test ./internal/search/...` (or touched packages)
- [ ] `npm run build` (if frontend touched)

## Deploy
- [ ] Deploy backend dev + smoke test
- [ ] ES mapping update via reindex
