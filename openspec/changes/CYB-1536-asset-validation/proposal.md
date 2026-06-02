# Proposal — CYB-1536 Asset Validation

## Problem
Pipeline deploy currently validates selected `asset_ids` by looping over `AssetRepository.Get`, while other run-like entry points such as `algo-runs.input_asset_ids` can accept unknown assets. This creates inconsistent behavior and inefficient validation for larger batches.

## Goals
- Add a shared batch asset existence validator.
- Use the validator in pipeline deploy / run creation paths.
- Use the validator in `POST /api/v1/algo-runs` for `input_asset_ids`.
- Return consistent `400 INVALID_ARGUMENT` errors with missing asset details.
- Preserve no-asset and filter-only run behavior.

## Non-Goals
- Changing asset search or asset list APIs.
- Requiring assets for every run; no-asset runs remain valid when explicit.
- Implementing first-class pipeline run persistence; that is handled by `CYB-1534-run-target-model`.

## Acceptance Criteria
- Explicit asset ID lists are checked in one batch query.
- Missing or deleted assets are rejected before creating a run/deployment/algo-run record.
- Error responses include the field name and missing asset IDs.
- Empty or omitted asset lists remain valid for no-asset/filter-only flows.
- Current deploy endpoints keep their existing request shape.
