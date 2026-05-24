# Spec delta — asset management (CYB-1017)

## ADDED: Asset provenance read API

**Given** an asset with `logical_asset_id` and multiple revisions
**When** a client calls `GET /api/v1/assets/{id}/provenance`
**Then** the response includes `version_history` ordered by `version` ascending
**And** each promote entry includes `promoted_at`, `by_run_id`, and `reason` when present in `version_promoted` events

**Given** the same logical asset family
**When** provenance is fetched for any revision `asset_id`
**Then** `revisions` lists all non-deleted rows for that `logical_asset_id`

## ADDED: Detail page version selector

**Given** provenance returns more than one revision
**When** the user opens asset detail
**Then** a version control is shown defaulting to the current revision
**And** selecting another revision navigates to that `asset_id`
