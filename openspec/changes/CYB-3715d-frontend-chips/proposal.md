# CYB-3715d: expose 7 flatten fields in the assets facet sidebar

## Context

The backend fully supports filter + facet on 7 flatten mirror columns after CYB-3715 (#526 / #527 / #528 / #530). The AssetsFacetSidebar still lists only pre-CYB-3715 fields (`mcap.vendor_id`, `mcap.scene_id`), so:

- Users can't discover `camera_model`, `source_platform`, etc. as facet chips
- Direct-column filters require typing the full field name into the free filter dropdown

## Change

Frontend-only. No API contract change.

1. `useAssetsDiscoveryReducer.ts` `ASSET_DISCOVERY_FACETS` — request 4 facet-able flatten fields (`camera_model`, `data_source`, `collection_method`, `source_platform`). Backend PGSupportedFacetFields already knows how to answer.
2. `AssetsFacetSidebar.tsx` `AGG_KEY_MAP` — 4 self-referential entries so `fieldCounts[<field>]` picks up the new agg buckets returned by `/queries/run`.
3. `AssetsFacetSidebar.tsx` `GROUP_FIELDS.capture` — 4 facet-able + 3 uuid input-only fields declared.
4. `AssetsFacetSidebar.tsx` capture group JSX — 4 CheckboxFacet chips (guarded by non-empty bucket presence, matches the existing `env` pattern) + 3 InputFacet for direct-column UUIDs.

UUID chips use InputFacet since UUIDs are filter-only (too high cardinality for facet chip rendering — same rationale as PGSupportedFacetFields).

## Non-goals

- Reworking the free filter dropdown (a separate component with its own scope + bit-rot risk)
- Adding these fields to the assets list table columns
- Browser regression test — frontend tests are off CI (per `project_frontend_tests_bitrot.md`); manual browser verification tracked as post-merge action
