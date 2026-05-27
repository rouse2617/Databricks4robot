# Spec Delta — 资产模型扩展 Phase 1

## Given
- Assets table with existing asset_types (raw_mcap, segment, clip, frame, task, action, derived_asset)
- asset_relations with CHECK constraint on relation_type (split_from, derived_from, contains, etc.)
- Write validation via hardcoded switch in usecase layer
- ES index with current asset_type-specific mappings

## When Phase 1 is implemented
- New `dataset` assets can be created with optional metadata fields
- New `annotation_result` assets can be created linked to parent assets
- asset_relations accepts `annotated_from`, `materialized_from`
- Schema registry drives validation instead of switch statements

## Then
- `POST /api/v1/assets` with `asset_type=dataset` succeeds with valid metadata
- `POST /api/v1/assets` with `asset_type=annotation_result` succeeds
- `GET /api/v1/asset-types/dataset/schema` returns JSON Schema
- Existing segment/clip creation still works (backward compatible)
