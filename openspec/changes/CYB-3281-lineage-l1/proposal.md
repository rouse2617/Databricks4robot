# Proposal — CYB-3281 (Asset lineage L1 minimal)

## Why

CYB-3268 made action a first-class asset, but `GET /assets/:id/lineage` shows no
asset-type lineage: `buildLineageResponse` runs 4 SQLs and **never reads the
`asset_relations` edge table nor `assets.parent_asset_id`**. The data is written
(parent_asset_id + asset_relations edges), it's just not read back. Two adjacent
gaps: relation `metadata` is always `'{}'` (never written), and
`RegisterOutput` writes `pipeline_output` edges that the CHECK constraint
rejects → silently dropped.

## What Changes

### Modified Capabilities
- **asset lineage read** — `buildLineageResponse` adds the immediate **parent
  asset** to `upstream` and a **`downstream.children`** list, by reading
  `assets.parent_asset_id` / `asset_relations`.
- **relation metadata** — `AssetRelationWriter` gains
  `InsertRelationWithMetadata(...)`; callers pass caller context (split_method /
  run_id / deployment_id …) into `asset_relations.metadata`.
- ~~**CHECK fix**~~ — **DEFERRED** (user 2026-07-10: "pipeline_output 后面再说").
  No migration here; `RegisterOutput`'s `pipeline_output` edges stay as-is (still
  rejected by the CHECK) until a follow-up. **This change touches no schema.**

### Design decisions
- **upstream is only-ADD, not reshaped** (avoids frontend regression): the
  existing top-level mcap fields (`mcap_file_id` / `mcap_uri` / `ingest_state`)
  **stay put** (current `LineageTab` + CYB-3279 read them there). We *add*
  `asset_id` / `asset_type` / `root_asset_id` / `start_timestamp_ns` /
  `end_timestamp_ns` (the parent asset) as new sibling keys. No `upstream.mcap`
  nesting. Acceptance #1 (`upstream.asset_id = mcap_id`, `asset_type='raw_mcap'`)
  holds because a segment's parent is the raw_mcap asset (id == mcap_file_id).
- **`downstream.children` is additive**: existing `algo_results` / `deliveries`
  / `eval_results` stay; `children[]` is a new sibling key.
- **Constraint name is `chk_relation_type`** (the issue guessed
  `asset_relations_relation_type_check`; the real name in the baseline is
  `chk_relation_type`). Migration uses `DROP CONSTRAINT IF EXISTS chk_relation_type` + re-add.
- **Interface adds one method** (`InsertRelationWithMetadata`); `InsertRelation`
  stays as a thin `metadata=nil` wrapper. All mocks updated in the same change.

## Impact
- `backend/internal/handlers/asset/lineage_response.go` — +~70 lines (2 queries, only-add).
- `backend/internal/repository/asset_relation_writer.go` — +1 interface method.
- `backend/internal/postgres/repos.go` — real impl gains metadata; `InsertRelation` becomes a wrapper.
- caller: `usecase/asset/usecase.go` (CreateChildAsset) passes `{split_method, split_run_id}` metadata on its derived_from/split_from edges (already whitelisted). `RegisterOutput` (pipeline_output) left unchanged — deferred. `versioning.go` uses `InsertRevisionOf` (unchanged).
- 2 test mocks: `usecase/pipeline/usecase_crud_test.go`, `usecase/asset/versioning_test.go` — add the new interface method.
- **No migration** (CHECK fix deferred).
- API contract: `api/openapi.yaml` + `docs/review/api-guide.md` document the new `/lineage` upstream/downstream fields.

## Scope
- **In (L1)**: lineage read (parent + children), relation metadata write, CHECK fix, tests.
- **Out**: L2 (relation-type APIs, root_asset_id backfill, RegisterOutput sync parent) → CYB-3282; L3 (graph query, time dimension, LineageTab react-flow upgrade) → CYB-3283; Neo4j; materialized lineage columns/ES arrays.

## Success Criteria
- [ ] `GET /assets/<seg>/lineage` upstream has `asset_id`(=mcap id) + `asset_type='raw_mcap'`, **and** still has top-level `mcap_file_id`/`mcap_uri`/`ingest_state`.
- [ ] `GET /assets/<mcap>/lineage` `downstream.children` lists its segments.
- [ ] `GET /assets/<seg>/lineage` `downstream.children` lists its actions/frames.
- [ ] `asset_relations.metadata` is no longer always `'{}'` for new child (derived_from/split_from) edges.
- [ ] `go build`/`vet`/touched tests pass; frontend upstream mcap card unaffected.
- [ ] (deferred) pipeline_output CHECK fix — separate follow-up.
