# Proposal — CYB-3203: Asset metadata endpoint for UI

## Why
`assets.metadata` and `mcap_files.metadata` (both JSONB) hold rich per-asset
context that grace_sync writes and grace operations rely on (grace_video_snapshot,
storage_meta.gcs / aliyun, collection_meta, process_info, video_info). The
existing `GET /api/v1/assets/:id` response **omits** these JSONB fields
(`json:"metadata,omitempty"` + empty-maps drop in JSON), so the UI cannot see
anything below the curated top-level fields. Grace operators need that
information to triage assets.

## What Changes
### New capability
- **`GET /api/v1/assets/:id/metadata`** — returns the raw `assets.metadata`
  and `mcap_files.metadata` JSONB plus a small set of convenience fields
  (grace_video_snapshot, storage, collection, process_info, video_info) for
  the convenience of the UI.

### Decisions
1. **New endpoint** (not modifying `GET /assets/:id`) so the existing asset
   contract stays stable. The UI fetches the new endpoint on demand from the
   asset detail page.
2. **Auth**: same as `GET /assets/:id` — any `Authenticate`-passing principal
   (no additional `RequireScope`). Follows the existing read-asset policy.
3. **Response shape** (full):
   ```json
   {
     "asset_id": "...",
     "segment_locator": "...",
     "lifecycle_state": "...",
     "asset_metadata": { ... },          // raw assets.metadata
     "mcap_metadata": { ... },            // raw mcap_files.metadata
     "grace_video_snapshot": { ... },     // = asset_metadata.grace_video_snapshot or null
     "storage": { "gcs": {...}, "aliyun": {...} },   // from mcap_metadata.storage_meta.*
     "collection": { ... },               // = mcap_metadata.collection_meta
     "process_info": { ... },             // = mcap_metadata.process_info
     "video_info": { ... }                // = mcap_metadata.video_info
   }
   ```
4. **No truncation** in v1 — observed sizes are < 5KB; can revisit if it grows.
5. **UI** uses a single Antd `<Collapse>` panel ("高级 / 元数据") on the
   asset detail page, **UX-1.5**: collapsed by default; when expanded, renders
   a `<JsonViewer collapsed={1}>` so users see only the level-1 keys
   (`asset_metadata` / `mcap_metadata` / `storage` / `processing` / ...) and
   drill down on demand. **No tabs** — the JsonViewer's level-1 keys serve as
   the navigation, matching the natural mental model with minimal code.

## Non-goals
- Mutating metadata via API (out of scope).
- Surfacing `mcap_files.process_state` separately (already a sub-tree of
  `mcap_metadata`; UI can navigate).
- SDK / API-guide / smoke updates are **not optional** for this change
  (see API contract sync below).

## Impact
- **Backend**:
  - new handler: `GET /api/v1/assets/:id/metadata` (route in
    `backend/routes/routes.go`, handler near existing asset handlers).
  - reads `assets` + `mcap_files`; returns 404 if asset missing; 200 with
    `{mcap_metadata: null, ... convenience fields: null}` if no mcap row.
- **OpenAPI / API-guide / SDK / smoke**: required by AI-RULES
  (api/openapi.yaml + docs/review/api-guide.md + sdk client + unit tests +
  scripts/api-guide-smoke.sh at minimum).
- **Frontend**:
  - new typed client in `Frontend/src/api/` (or extend existing).
  - single `<Collapse>` panel on the asset detail page wrapping one
    `<JsonViewer collapsed={1}>`. **No tabs** (UX-1.5; see Decision 5).
  - **New dep**: `react-json-view` (or ~30-line custom `<JsonNode>` if team
    prefers zero-dep). Declare in PR description per AI-RULES.

## Success Criteria
- [ ] `GET /api/v1/assets/91022781/metadata` returns 200 with `asset_metadata`
      (grace_video_snapshot tree) and `mcap_metadata` (storage_meta, process_info, etc.)
      for a real dev asset.
- [ ] Missing asset → 404; asset without mcap_files row → 200 with
      `mcap_metadata: null` (and convenience fields null).
- [ ] Existing `GET /api/v1/assets/:id` contract unchanged (no new fields in
      top-level).
- [ ] OpenAPI updated, SDK method exposed + unit-tested, smoke happy + error path
      passes locally.
- [ ] Frontend collapsible section renders correctly on the asset detail page.

## Open questions (deferred to implementation, not blocking this spec)
- Truncation threshold if metadata balloons (revisit after first prod sighting).
- ETag / conditional GET (current `GET /assets/:id` doesn't have one; mirror).
- Permission tier — should this endpoint require `assets:read` scope? Current
  `GET /assets/:id` requires only authentication, so v1 mirrors that.
