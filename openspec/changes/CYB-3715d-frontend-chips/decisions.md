# CYB-3715d decisions

## Why CheckboxFacet for the 4 text fields

The 4 fields are enum-like (camera families, data source values, platform names) and have low cardinality on dev (`camera_model`: 3 buckets, `source_platform`: 1 bucket). CheckboxFacet is the discoverable UX — user sees exactly what values exist. Matches the `env` pattern added in CYB-3297 Phase B.

## Why InputFacet (not CheckboxFacet) for the 3 UUID fields

UUIDs are high-cardinality (potentially one bucket per device). Rendering N-hundred UUID checkboxes in a sidebar is unusable. InputFacet lets users paste a known UUID they got from elsewhere (mcap detail page, external tool). This is the same rationale that keeps them out of `PGSupportedFacetFields` on the backend.

## Why not add these to the free filter dropdown

The free filter dropdown lives in a separate component (AssetsSearchBar + AddFilterPopover). It has its own field registry with its own bit-rot risk (frontend tests off CI per `project_frontend_tests_bitrot.md`). This PR keeps blast radius minimal and lands the discoverable UI first. A follow-up can extend the dropdown once we see how users actually use the sidebar chips.

## Why not manual browser verification pre-merge

Cron cycle time budget. Post-merge verification tracked in tasks.md — depends on #530 merged + reindex complete for keyword-mode facets. Structured mode already works via PG path.

## Why keep both `scene_id` and `mcap.scene_id`

Different semantics:
- `scene_id` — direct assets column (CYB-3715 mirror, addresses assets.scene_id)
- `mcap.scene_id` — EXISTS subquery on mcap_files.scene_id (existing behavior)

On backfilled rows they return the same result, but a caller writing an asset with a different mcap linkage (or no mcap) will see them diverge. Both are legitimate query paths. UI shows both to make the distinction visible; label "场景 UUID" vs "场景 ID" hints at it.
