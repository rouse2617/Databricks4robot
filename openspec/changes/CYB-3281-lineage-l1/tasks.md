# Tasks — CYB-3281

## Context files
- `backend/internal/handlers/asset/lineage_response.go:20` — `buildLineageResponse`
- `backend/internal/repository/asset_relation_writer.go` — `AssetRelationWriter` interface
- `backend/internal/postgres/repos.go:55` — `InsertRelation` impl (real INSERT)
- callers: `usecase/asset/usecase.go:1138` (CreateChildAsset), `usecase/pipeline/usecase.go:5685,5690` (RegisterOutput)
- mocks: `usecase/pipeline/usecase_crud_test.go`, `usecase/asset/versioning_test.go`
- `migrations/20260708104125_baseline_from_dev.sql:428` — `chk_relation_type` (current allowed set, no `pipeline_output`)

## Implementation
### 1.3 migration (CHECK fix) — DEFERRED (user 2026-07-10: "pipeline_output 后面再说")
- [~] no migration in this change; `pipeline_output` CHECK fix + RegisterOutput metadata → separate follow-up. **This PR touches no schema.**

### 1.2 relation metadata
- [ ] [backend] `asset_relation_writer.go` — add `InsertRelationWithMetadata(ctx, parentID, childID, relationType, runID string, metadata map[string]any) error`
- [ ] [backend] `repos.go` — real impl → `InsertRelationWithMetadata` (INSERT writes `metadata` col, `$5::jsonb`); `InsertRelation` = wrapper (`metadata=nil` → `'{}'`)
- [ ] [backend] caller passes metadata: `CreateChildAsset` (`{split_method, split_run_id}`) on derived_from/split_from edges; `RegisterOutput` unchanged (deferred); `InsertRevisionOf` unchanged
- [ ] [backend] mocks (`usecase_crud_test.go`, `versioning_test.go`) — add `InsertRelationWithMetadata`

### 1.1 lineage read (only-ADD)
- [ ] [backend] `lineage_response.go` upstream — keep mcap fields top-level; ADD parent asset via `assets self JOIN assets parent ON parent.asset_id=self.parent_asset_id` → merge `asset_id/asset_type/root_asset_id/start/end` into the same `upstream` gin.H
- [ ] [backend] `lineage_response.go` downstream — ADD `children[]` (`SELECT asset_id, asset_type, parent_asset_id, root_asset_id, metadata->>'import_batch' WHERE parent_asset_id=$1 AND is_deleted=FALSE ORDER BY asset_type, asset_id LIMIT 50`) alongside algo/deliveries/eval

### tests
- [ ] [backend] `lineage_response_test.go` (or handler_test) — parent in upstream (mcap fields intact) + children listed; fake pg/pgq

## API contract sync
- [ ] `api/openapi.yaml` — `/assets/{id}/lineage` response: upstream +{asset_id,asset_type,root_asset_id,...}, downstream +children[]
- [ ] `docs/review/api-guide.md` — lineage §: document new upstream parent + downstream.children (note mcap fields unchanged)
- [ ] SDK: no change (dict passthrough)

## Verification
- [ ] `cd backend && go build ./... && go vet ./... && go test ./internal/handlers/asset/... ./internal/postgres/... ./internal/usecase/asset/... ./internal/usecase/pipeline/...`
- [ ] no migration (CHECK fix deferred)

## Deploy verification (post-merge, PR→CICD; no migration)
- [ ] backend: `GET /assets/6EDE33F6/lineage` → downstream.children lists its 13 segments; a segment's lineage upstream has asset_id + asset_type='raw_mcap' + still mcap_file_id
- [ ] frontend regression: LineageTab 上游 mcap card still renders (top-level fields untouched) — Chrome MCP

## PR
- [ ] title: `feat(backend): asset lineage L1 — read asset_relations + write relation metadata (cyb-3281)`
- [ ] body: Linear/OpenSpec/API-sync/BREAKING?(no — additive)/deploy evidence; note pipeline_output CHECK fix deferred, no migration
