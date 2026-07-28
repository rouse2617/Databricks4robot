# Backfill — dispatch history & per-asset listing (delta)

## ADDED: filter batches by creator

### Requirement: `GET /api/v1/backfill` accepts optional `createdBy`

The batch list endpoint SHALL accept an optional `createdBy` query parameter and, when present, return only jobs whose `created_by` equals it.

#### Scenario: filter by subscription task
- **Given** three batches exist, two created by `subscription-task:sub_A` and one by `sub_B`
- **When** a client GETs `/api/v1/backfill?createdBy=subscription-task:sub_A`
- **Then** the response contains exactly the two `sub_A` batches, ordered by `created_at` descending

#### Scenario: omitted filter is unchanged
- **Given** any set of batches
- **When** a client GETs `/api/v1/backfill` with no `createdBy`
- **Then** all batches are returned (existing behavior, response shape `{items: [...]}`)

## ADDED: list a batch's per-asset items

### Requirement: `GET /api/v1/backfill/:id/items` returns per-asset rows

The endpoint SHALL return the asset-level items of a batch, each with its asset id and processing status.

#### Scenario: batch with items
- **Given** batch `batch_X` has two items (assets `a1` completed, `a2` failed)
- **When** a client GETs `/api/v1/backfill/batch_X/items`
- **Then** the response is `{items: [...]}` with two entries carrying `assetId` and `status` (and `pipelineRunId` when dispatched)

#### Scenario: unknown or item-less batch
- **Given** batch id `batch_missing` has no items
- **When** a client GETs `/api/v1/backfill/batch_missing/items`
- **Then** the response is `200` with `{items: []}` (not a 404)

#### Scenario: empty id
- **When** a client GETs `/api/v1/backfill//items` (empty id)
- **Then** the response is `400` with code `INVALID_ARGUMENT`
