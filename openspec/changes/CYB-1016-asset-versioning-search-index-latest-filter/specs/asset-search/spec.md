# Spec delta — asset search (CYB-1016)

## ADDED: Version fields in search index

**Given** an asset row with `logical_asset_id` set
**When** the search indexer builds an ES document
**Then** the document includes `logical_asset_id`, `revision`, and `is_current`

**Given** a legacy asset without `logical_asset_id`
**When** the search indexer builds an ES document
**Then** the document omits `logical_asset_id` and remains discoverable under the default current-only filter

## ADDED: Default current-only query results

**Given** multiple revisions for one logical asset
**When** a client calls `POST /api/v1/queries/run` without `include_history`
**Then** only rows with `is_current=true` (or legacy NULL `is_current`) are returned

**Given** the same logical asset family
**When** a client calls `POST /api/v1/queries/run?include_history=true`
**Then** all non-deleted revisions may appear in results

## ADDED: Prior revision re-index on promote

**Given** a successful version promote with a prior current `asset_id`
**When** `version_promoted` is recorded for the new revision
**Then** an `asset_updated` event is also recorded for the prior `asset_id` so Elasticsearch reflects `is_current=false` on the old revision
