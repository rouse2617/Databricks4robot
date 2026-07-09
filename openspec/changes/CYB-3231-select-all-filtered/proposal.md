# CYB-3231 Make "select all filtered results" actually select

## Problem
Clicking "选择全部 N 条筛选结果" does nothing observable: `SELECT_ALL_FILTERED`
only set `selectionState.mode = "all_filtered_results"` but never populated
`selectedIds`. `selectedCount = selectedIds.size` and every bulk action uses
`selectedAssetIds = Array.from(selectedIds)`, and nothing reads the mode — so the
count stayed at the explicit count and delivery/pipeline/tag/export only acted on
the few explicitly-checked rows.

## Scope
- New reducer action `SET_SELECTED_IDS { ids }` → replaces `selectedIds` with the
  given ids (mode `explicit_rows`, or `none` when empty).
- `useAssetsDiscoveryReducer`: export `buildSelectAllIdsQueryRequest(queryState, limit)`
  (same where/sort as the list, `select: [asset_id]`, one large page) and a
  `SELECT_ALL_MAX` cap.
- `AssetsPage.onSelectAllFiltered`: async — fetch all filtered asset_ids (capped),
  dispatch `SET_SELECTED_IDS`, and toast success / a cap warning / error.

## Out of Scope
- Backend "operate by filter" batch endpoints (would remove the cap). Cap-based
  client fetch is the pragmatic first step.
- `algo_status` client-side filter caveat: the id fetch is backend-filtered, so an
  active `algo_status` chip may over-select (same limitation the old total had);
  documented, not solved here.

## Notes
- Cap protects against pulling unbounded id lists; when `total > cap`, select the
  first `cap` and warn the user that the remainder isn't selected.
