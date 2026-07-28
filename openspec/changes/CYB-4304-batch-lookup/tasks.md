# CYB-4304 tasks

- [ ] `components/batch-lookup/BatchAssetLookup.tsx` — preset-driven component. Owns: input textarea, id_type radio, parse+dedup preview, extra filters slot, submit, request/response wire, stats bar, histogram slot, results table, missing/filtered-out sections, CSV export
- [ ] `components/batch-lookup/types.ts` — `BatchLookupPreset<Item, Filters>`, `StatEntry`, `BucketRow` types + `parseIdBlob` helper moved here so it's reachable from any preset
- [ ] `pages/presets/durations.ts` — durations preset config (fetch → assetsApi.lookupDurations, columns, csv, histogram bucketization, extraStats)
- [ ] `pages/AssetDurationLookup.tsx` — reduced to `<BatchAssetLookup preset={durationsPreset} />` (keeps default-export contract so lazy-import in AssetsWorkbenchPage keeps working)
- [ ] `components/batch-lookup/BatchAssetLookup.test.tsx` — parse edge cases, cap-disabled submit, empty-response missing-section render
- [ ] `pages/AssetDurationLookup.test.tsx` — existing tests still pass (as integration for the durations preset)

## Verification
- [ ] `npx tsc --noEmit` clean (modulo pre-existing DeployPanel error)
- [ ] `npx vitest run` on touched files
- [ ] `npx vite build` clean
- [ ] manual dev walkthrough: durations preset via `/assets?view=durations` renders identically to CYB-4294

## Ship
- [ ] Commit, push, PR base=dev
- [ ] Merge, deploy, dev smoke
