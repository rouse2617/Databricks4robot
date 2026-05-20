# Tasks — CYB-985

## Context files

- `services/mcap-preview/internal/manifest/`
- `services/mcap-preview/internal/server/segment.go`
- `Frontend/src/hooks/assets/useAssetPreview.ts`
- `Frontend/src/hooks/assets/useAssetsDiscoveryReducer.ts`
- `Frontend/src/components/assets/PreviewPlayer.tsx`
- `Frontend/src/components/asset-detail/AssetPreviewHero.tsx`
- `Frontend/src/pages/AssetDetailPage.tsx`
- `Frontend/src/components/assets/AssetQuickPreviewPane.tsx`

## Tasks

- [x] manifest: `DetectTopicCodec` + `sources[].codec` in `buildPreviewSources`
- [x] server: reuse manifest codec helper from segment `pickTopic`
- [x] frontend: parallel `getPreviewManifest`, enrich segment URL (window, grace_token)
- [x] frontend: `prewarmPreview` on HEVC source
- [x] frontend: PreviewPlayer error mapping + tests
- [x] frontend: detail hero source picker + `preview_source` URL sync + refetch on change
- [x] frontend: sidebar source picker `showSearch` for topic filter
- [x] docs: update `services/mcap-preview/README.md`
- [x] manifest: stable sort + consecutive `live_topic_N` (`sources_test.go`)
- [x] frontend: detail page source-only reload (no false new-asset reset on `preview_source` change)
- [x] dev: deploy + verify on `niUShaN6` (front/side/down direct links, invalid id redirect)
