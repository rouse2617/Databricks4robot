# Tasks — CYB-1032

## P0 — Critical

- [x] **Resolve dual `:root` conflict** — removed conflicting color definitions from `design-tokens.css`, kept non-conflicting tokens (typography, spacing, shadows)
- [x] **Add Tailwind color tokens** — extended `tailwind.config.js` with 8 semantic color classes (primary, success, warning, error, text-secondary, surface, border, primary-light)
- [x] **Fix LogicalAssetId contrast** — `#94A3B8` → `text-text-secondary` (3.05:1 → 4.63:1)
- [x] **Fix `#999` text contrast** — replaced in CmdKSearch, AlgoMatrixGrid, AlgoStatusPopover, LineageTab, AddFilterPopover, SavedViewSelector (2.8:1 → 4.63:1)
- [x] **Fix AlgoStatusCell contrast** — replaced hardcoded status colors with project palette (#16a34a/#dc2626/#d97706)
- [x] **Fix heading hierarchy** — `AssetDetailPage.tsx` h4 → h1

## P1 — High priority

- [x] **Fix touch targets in VersionProvenanceTab** — removed `p-0 h-auto` from "查看此版" and "在血缘Tab查看完整" buttons
- [x] **Fix VersionHistoryBanner button size** — `size="small"` → `size="middle"`
- [x] **Fix tab loading counts** — show `(...)` instead of `(0)` during data fetch

## P2 — Design system

- [x] **Replace VersionHistoryBanner with Alert** — 67-line custom component → Ant Design `<Alert type="warning">`
- [x] **Replace hardcoded hex in VersionControl** — 13 values → Tailwind semantic classes
- [x] **Replace hardcoded hex in VersionProvenanceTab** — 5 values → Tailwind semantic classes + `<ol role="list">`
- [x] **Replace hardcoded hex in LogicalAssetId** — 3 values → Tailwind semantic classes
- [x] **Replace hardcoded hex in AssetDetailPage** — pipe separator → `text-border`

## Verification

- [x] Tier L: `cd Frontend && npm run build` — passed
- [x] Chrome DevTools MCP: visual verification on dev deploy — 7 pages tested, 0 console errors
- [x] Contrast check: all text meets WCAG AA (4.5:1)
- [x] Touch targets: all interactive elements ≥ 44px

## Deploy Record

| Service | Image | Revision | URL |
|---------|-------|----------|-----|
| frontend-dev | `cyber-databrew-frontend:60ed161` | `cyber-databrew-frontend-dev-00186-2zk` | https://cyber-databrew-frontend-dev-234851712830.us-central1.run.app |
