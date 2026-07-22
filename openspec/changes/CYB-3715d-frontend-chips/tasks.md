# CYB-3715d tasks

- [x] `useAssetsDiscoveryReducer.ts` — request 4 facet-able flatten fields in `ASSET_DISCOVERY_FACETS`
- [x] `AssetsFacetSidebar.tsx` — 4 entries in `AGG_KEY_MAP`
- [x] `AssetsFacetSidebar.tsx` — extend `GROUP_FIELDS.capture` with 7 fields
- [x] `AssetsFacetSidebar.tsx` — capture group JSX: 4 CheckboxFacet + 3 InputFacet
- [x] `npx tsc --noEmit` — clean
- [x] `AssetsFacetSidebar.test.tsx` — 33/33 pass
- [x] `useAssetsDiscoveryReducer.test.ts` — 3 pre-existing failures (bit-rot, confirmed on origin/dev without my changes)
- [x] `npx vite build` — clean

## Post-merge

- Manual browser verification: open /assets, expand "采集" group, verify 4 chip groups appear (only when populated) + 3 UUID input rows visible
- Depends on: PR #530 merged + `admin/search/reindex` complete so `mode=keyword` returns facets. `mode=structured` (default) already works with just the backend PG path.
