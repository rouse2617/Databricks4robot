# CYB-4333: consolidate batch-lookup tabs behind a single "批量查询" hub

## Context

After CYB-4304 (durations) and CYB-4306 (costs) landed, `/assets` grew from 1 → 3 top tabs. CYB-4305 (lineage) will make 4. Planned collectors/deliveries follow. Every new preset means a new tab — the strip gets crowded and navigation-heavy for what is fundamentally "one kind of workflow with different fields".

## Change

Collapse batch lookups behind a single "批量查询" tab that hosts a preset picker. The lookup UI itself stays exactly the same per preset.

### Tab strip (outer)

Before: `资产列表 | 时长批量查询 | 成本查询`
After:  `资产列表 | 批量查询`

### Inside the 批量查询 tab

Top row: `Segmented` control listing all presets. Selecting one renders the corresponding `BatchAssetLookup`. Default is the first preset in the registry (currently `durations`).

### URL

- `/assets` → 资产列表 (unchanged)
- `/assets?view=lookup` → 批量查询, default preset
- `/assets?view=lookup&preset=<key>` → 批量查询 + specific preset

Old URLs redirect (Navigate replace):
- `/assets?view=durations` → `/assets?view=lookup&preset=durations`
- `/assets?view=costs`     → `/assets?view=lookup&preset=costs`

### Preset registry

`Frontend/src/pages/presets/index.ts` exports the map: `BATCH_LOOKUP_PRESETS = { durations, costs }`. Adding a preset = one import + one entry.

### Files touched

- `AssetsWorkbenchPage.tsx` — VIEW_KEYS from 3 → 2, add old-URL redirects
- New `BatchLookupPage.tsx` — reads `?preset=`, renders Segmented + BatchAssetLookup
- New `presets/index.ts` — registry
- Delete `AssetDurationLookup.tsx` and `AssetCostsLookup.tsx` (5-line shims; imports of `<BatchAssetLookup preset={durationsPreset} />` inline into the hub)
- Update `AssetsWorkbenchPage.test.tsx` — 2-tab assertion + redirect assertion
- New `BatchLookupPage.test.tsx` — preset switch, URL sync both directions, unknown-preset fallback

## Non-goals

- Changing any preset's fields, histogram, or CSV format
- Sidebar entry — still `/assets`
- Preset picker becoming a dropdown — Segmented handles 2-6 items cleanly; revisit at >6

## Compatibility

- Any bookmarks / external links to `/assets?view=durations|costs` redirect (URL updates via `<Navigate replace />`), so they don't burn as dead links.
- No backend contract change.
