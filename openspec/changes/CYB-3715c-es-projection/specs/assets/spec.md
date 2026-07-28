# ES doc spec delta — CYB-3715c

## Top-level fields projected from Asset

`internal/searchindex/builder.go` `Builder.Build` now emits the following as top-level string properties **when the Asset struct field is non-empty**:

| field | source (Asset struct) |
|---|---|
| `camera_model` | `a.CameraModel` |
| `device_id` | `a.DeviceID` |
| `collector_id` | `a.CollectorID` |
| `scene_id` | `a.SceneID` |
| `data_source` | `a.DataSource` |
| `collection_method` | `a.CollectionMethod` |
| `source_platform` | `a.SourcePlatform` |

Coexists with (not replacing) the existing nested `mcap.<col>` object from CYB-3297 Phase C. Nested mcap.* stays for callers of the mcap.<col> filter syntax; top-level is what filter/facet compilers hit after CYB-3715.

## Facet whitelist

`internal/elasticsearch/query_ir.go` `facetFieldPath` accepts 4 more direct-column facet fields (matching PG `PGSupportedFacetFields`):

- `camera_model`
- `data_source`
- `collection_method`
- `source_platform`

UUID fields (device_id / collector_id / scene_id) remain filter-only; too high-cardinality for terms aggregation.

## Additivity

- Empty struct field → property absent from doc (no empty-string keyword buckets)
- Docs indexed pre-CYB-3715c → require `admin/search/reindex` to populate the top-level fields. Filter still returns 0 from ES until reindex finishes; PG-only fallback via CYB-3384 planner covers this window.
