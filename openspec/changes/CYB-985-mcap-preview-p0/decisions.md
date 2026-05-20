# Decisions — CYB-985

## Stable `live_topic_N` assignment (manifest)

**Problem:** `buildPreviewSources` iterated MCAP channels from a Go map and used the loop index for `live_topic_N`. Map iteration order is non-deterministic, so the same `?preview_source=live_topic_3` could point at front, side, or down across requests. Direct links and detail-page URL sync appeared broken.

**Decision:** Sort candidate topics by `topicPreferenceScore` (desc), then topic name (asc), before assigning consecutive `live_topic_0`, `live_topic_1`, … Use a dedicated `sourceIndex` counter so skipped non-video topics do not leave gaps in ids.

**Consequence:** Bookmarked `preview_source` values are stable for the **same MCAP file**. Legacy ids from pre-fix dev (e.g. `live_topic_3` for front on some loads) are invalid after deploy; users should use manifest ids from `GET …/manifest` (typical 3-camera asset: `0`=front, `1`=side, `2`=down).

**Known limitation:** `live_topic_N` is still index-based, so adding/removing topics in a re-recorded MCAP shifts ids. P1 should migrate to topic-derived slugs (e.g. `front_image_raw_compressed`) so ids survive MCAP changes.

## Detail page load vs source change

**Problem:** Effect cleanup reset `loadedPageKeyRef`, so a `preview_source` URL change was treated as a new asset and triggered full page reload races.

**Decision:** Track only `loadedAssetIdRef` for full-page reset; on same asset, `preview_source` changes call `fetchPreviewForSource` only. Cancel stale fetches via `previewFetchGenRef`.
