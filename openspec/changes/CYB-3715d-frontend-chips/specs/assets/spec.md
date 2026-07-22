# Sidebar spec delta — CYB-3715d

## AssetsFacetSidebar capture group

Adds 7 fields alongside the pre-existing `env` + `mcap.<col>` set:

| field | control | source of options |
|---|---|---|
| `camera_model` | CheckboxFacet | aggregation buckets, sorted by count desc |
| `data_source` | CheckboxFacet | aggregation buckets, sorted by count desc |
| `collection_method` | CheckboxFacet | aggregation buckets, sorted by count desc |
| `source_platform` | CheckboxFacet | aggregation buckets, sorted by count desc |
| `device_id` | InputFacet | free-typed UUID |
| `collector_id` | InputFacet | free-typed UUID |
| `scene_id` | InputFacet | free-typed UUID (distinct from `mcap.scene_id`) |

CheckboxFacet rendering is guarded by `fieldCounts.<field>` non-empty — matches the existing `env` pattern so buckets don't render as empty containers before the first aggregation lands.

## Facet request set

`ASSET_DISCOVERY_FACETS` in `useAssetsDiscoveryReducer.ts` requests 4 additional facet fields. Backend `PGSupportedFacetFields` (CYB-3715 planner update) + ES facet whitelist (CYB-3715c) both know how to answer.

## Filter identity

Sidebar filter chips submit the field name verbatim (`camera_model`, `device_id`, ...). Backend filter path (`filter/assets_fields.go` after CYB-3715a) resolves them to direct columns on `assets`. No new schema wiring needed.
