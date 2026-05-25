# CYB-1201: Fix asset handler bugs

## Summary

Fix 7 medium-severity bugs found in asset handler code review.

## Bugs

### Bug 1: BatchGet accepts duplicate asset IDs
- **Problem**: Duplicate `asset_ids` produce duplicate items in response
- **Fix**: Deduplicate `req.AssetIDs` before calling usecase

### Bug 2: ListDeliveries missing assetID validation
- **Problem**: Uses `c.Param("id")` without validation; invalid ID returns empty results
- **Fix**: Use `handlers.RequirePathAssetID(c)` like other asset handlers

### Bug 3 & 4: GetLineage / GetProvenance silently swallows lineage errors
- **Problem**: `body, _ := h.buildLineageResponse(...)` discards errors
- **Fix**: Check error from buildLineageResponse, log and include partial data indication

### Bug 5: ListDeliveries pagination never returns next_token
- **Problem**: Always `""` even when there are more results
- **Fix**: Compute next_token based on whether end < len(ids)

### Bug 6: ListDeliveries returns raw IDs instead of delivery objects
- **Problem**: Inconsistent with other list endpoints that return full objects
- **Fix**: This is by design (minimal endpoint), mark as known limitation

### Bug 7: AlgoHandler Start/Finish missing assetID validation
- **Problem**: Uses `c.Param("id")` without validation
- **Fix**: Add assetID validation similar to other handlers

## Verification

1. `go build ./...` passes
2. `go test ./internal/handlers/asset/...` passes
3. Deploy dev + curl verify
