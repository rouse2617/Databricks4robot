# CYB-4333 tasks

## Registry
- [ ] `Frontend/src/pages/presets/index.ts` — export `BATCH_LOOKUP_PRESETS` map keyed by preset id (`durations`, `costs`), plus `BatchLookupPresetKey` type + `DEFAULT_PRESET_KEY = "durations"` + a `presetOptions` array `[{ key, label, icon }]` for the Segmented control.

## Hub page
- [ ] `Frontend/src/pages/BatchLookupPage.tsx` — reads `?preset=` from URL, defaults to `DEFAULT_PRESET_KEY`, renders:
  - Top-of-page `Segmented` control with `presetOptions`
  - `<BatchAssetLookup preset={BATCH_LOOKUP_PRESETS[activeKey]} />` below it
  - URL sync via `useSearchParams` (setSearchParams with `{ replace: true }` — mirror `AssetsWorkbenchPage` pattern)
  - Unknown preset key falls back to `DEFAULT_PRESET_KEY` (do NOT throw; UX defense)

## Workbench
- [ ] `Frontend/src/pages/AssetsWorkbenchPage.tsx`:
  - Reduce `VIEW_KEYS` to `["list", "lookup"]`
  - Tab labels: `资产列表 / 批量查询` (icons: TableOutlined / SearchOutlined)
  - Add compatibility redirect: when `view` is `durations` or `costs`, `<Navigate to="/assets?view=lookup&preset=<v>" replace />` before rendering tabs
  - Update the lazy import: drop `AssetDurationLookup` + `AssetCostsLookup`, add `BatchLookupPage`
  - Update conditional render: `view === "lookup" ? <BatchLookupPage/> : <AssetsPage/>`

## Cleanup
- [ ] Delete `Frontend/src/pages/AssetDurationLookup.tsx` (imports inline into `BatchLookupPage` via the registry)
- [ ] Delete `Frontend/src/pages/AssetCostsLookup.tsx`
- [ ] Grep for any remaining imports of those two paths; there should be none besides tests, which we update

## Tests
- [ ] `Frontend/src/pages/BatchLookupPage.test.tsx` — render happy path (default preset), Segmented switch updates URL, `?preset=costs` renders costs preset content, `?preset=bogus` falls back to durations without crashing
- [ ] `Frontend/src/pages/AssetsWorkbenchPage.test.tsx` — 2-tab render + click switches ?view; `?view=durations` redirects to `?view=lookup&preset=durations` (same for costs)
- [ ] existing `presets/durations.test.tsx` / `costs.test.tsx` stay green untouched
- [ ] Grep-verify that `AssetDurationLookup.test.tsx` (from CYB-4304) no longer exists OR is updated to hit `BatchLookupPage` — pick one; deletion is cleaner if the assertions were purely against the durations preset behavior (which now lives in `presets/durations.test.tsx`)

## Verify
- [ ] `npx tsc --noEmit` — clean modulo pre-existing DeployPanel error
- [ ] `npx vitest run <touched>` — MUST BE ALL GREEN
- [ ] `npx vite build` — clean

## Ship
- [ ] Commit + push + PR base=dev
- [ ] Merge, deploy, dev browser walkthrough: 2 tabs visible; ?view=lookup lands on durations; Segmented switches to costs; old URL redirect works
