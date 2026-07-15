# CYB-3228 Fix Action tab 404 (wrong path) + graceful no-preview

## Problem

### P1 — "Action 时间轴" tab is broken (real bug)
`Frontend/src/api/actions.ts` calls `GET/POST /assets/{id}/action-annotations`,
but no such backend route exists. The registered route is `/assets/:id/actions`
(`routes.go:370-373`, GET/POST/PATCH/DELETE); OpenAPI and api-guide both document
`/assets/{id}/actions` (Phase 1 shipped). So every call returns **404** (verified
on dev: `O2q7s1gG`), leaving the Action timeline tab empty and action creation
broken.

### P3 — no-transcode assets show a broken video + 404 noise
`AssetPreviewHero` renders a native `<video src=.../segment.mp4>` when
`manifest.mode === "mcap"`. For assets without a transcoded preview (e.g. a
raw_mcap), that mp4 404s (verified on dev: `l4CwLLCT`), leaving a broken player.

## Scope

- `actions.ts`: `action-annotations` → `actions` (2 occurrences) to match the
  registered backend route.
- `PreviewPlayer` / preview media panel: on video load error, fall back to the
  existing "暂无预览" state instead of a broken player.

## Out of Scope / Won't fix

- **Video progress-bar aria `valuetext` mojibake** ("已播放时间…" garbled): this is
  Chrome's native `<video>` control shadow-DOM localization — not our code (no
  `aria-valuetext`/that string in `src/`). Not fixable by us; documented only.
- If the segment.mp4 404 turns out to be the backend manifest reporting
  `mode=mcap`/`availability` incorrectly for no-preview assets, that backend fix
  is a separate follow-up; this change only makes the frontend degrade gracefully.
- No backend change, no schema change.
