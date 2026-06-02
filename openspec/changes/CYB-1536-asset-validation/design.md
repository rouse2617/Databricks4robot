# Design — CYB-1536 Asset Validation

## Current State
Pipeline deploy validates `asset_ids` in `pipeline.Usecase.Deploy` by calling `assetRepo.Get` once per ID. `algo-runs` persists `input_asset_ids` without strict existence validation.

## Shared Validator
Add a narrow batch method to the asset repository:
```go
FindExistingIDs(ctx context.Context, assetIDs []string) (map[string]struct{}, error)
```

Implement it with one Postgres query:
```sql
SELECT asset_id
FROM assets
WHERE asset_id = ANY($1)
  AND is_deleted = FALSE
```

The usecase validator should:
- Trim asset IDs.
- Reject blank IDs as invalid.
- De-duplicate only for lookup efficiency.
- Preserve first-seen request order when reporting missing IDs.
- Return no error for empty or omitted lists.

## Error Shape
Use standard error envelope behavior with details:
```json
{
  "code": "INVALID_ARGUMENT",
  "message": "input_asset_ids contain unknown assets",
  "details": {
    "field": "input_asset_ids",
    "missing_asset_ids": ["deadbeef"],
    "invalid_asset_ids": [""]
  }
}
```

For pipeline deploy compatibility, the field can be `asset_ids`.

## Consumers
- `pipeline.Usecase.Deploy` for `/api/v1/deploy`.
- `pipeline.Usecase.DeployByTemplateID` through `Deploy`.
- First-class `POST /api/v1/pipeline-runs` once the run model lands.
- `algorun.Usecase.Create` for `/api/v1/algo-runs`.

## Compatibility
Existing deploy request bodies keep `asset_ids`. Algo run request bodies keep `input_asset_ids`. No-asset runs and filter-only algo runs remain valid.

## Risks
- Adding a repository method requires updating all test mocks that implement `AssetRepository`.
- Soft-delete semantics must match the existing asset model.
- Very large asset batches may need a request size limit in a later pass.
