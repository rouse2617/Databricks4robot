# Proposal — CYB-985

## Why

Asset preview calls `segment.mp4` without mcap-preview manifest metadata, so users cannot pick camera topics, HEVC assets cold-start slowly, and window/timebase mismatches surface as opaque video failures. The asset **detail** hero previously had no source picker even though the list sidebar did.

## What Changes

### Modified Capabilities

- **asset-preview**: discovery **and detail** preview load manifest, show searchable multi-source picker, prewarm HEVC, pass asset window to segment URLs, clearer errors. Detail page syncs `preview_source` in the URL when switching cameras. Manifest emits **stable** `live_topic_N` ids so direct links and picker selection stay aligned with camera topics.

### New Capabilities

- None (uses existing `GET /preview/assets/:id/manifest` and `POST .../prewarm`).

## Impact

- **Affected code**: `services/mcap-preview/internal/manifest`, `Frontend/src/hooks/assets`, `Frontend/src/components/assets/PreviewPlayer.tsx`, `Frontend/src/components/asset-detail/AssetPreviewHero.tsx`, `Frontend/src/pages/AssetDetailPage.tsx`, `Frontend/src/components/assets/AssetQuickPreviewPane.tsx`
- **Dependencies**: none

## Scope

- **In scope**: P0 A–D in one PR; manifest `sources[].codec`; **stable** `live_topic_N` ids; frontend wiring for list sidebar **and** detail hero
- **Out of scope**: `packages/cyber-rosview` (phase 2); GCS page-size tuning (P1)

## Success Criteria

- [x] Preview fetch loads manifest in parallel with foxglove-source
- [x] Sidebar lists manifest video sources with codec label and search filter
- [x] Detail hero lists manifest video sources with codec label and search filter
- [x] Changing source on detail or sidebar updates `segment.mp4` URL including `topic`
- [x] Detail page persists selected source in `?preview_source=`
- [x] Invalid `preview_source` in URL is corrected to a valid manifest id (no refresh loop)
- [x] Direct `?preview_source=` navigation switches topic without stale video from prior source
- [x] Manifest assigns `live_topic_N` in stable topic order (not map iteration order)
- [x] HEVC selection triggers prewarm without blocking UI
- [x] segment URLs include `start_ns`/`end_ns` when hints provide window
- [x] Known preview error codes show actionable Chinese messages

## Dev verification (2026-05-20)

- **Branch:** `feat/CYB-985-mcap-preview-p0` (`b1503aa`)
- **Services:** `mcap-preview-dev-00016-7bn`, `cyber-databrew-frontend-dev-00171-57h`
- **Asset:** `niUShaN6` — `live_topic_0` front, `live_topic_1` side, `live_topic_2` down; invalid id redirects to `live_topic_0`
