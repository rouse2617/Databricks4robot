# Spec delta — assets

## ADDED

### Requirement: mcap-file top-level fields are mirrored onto the raw_mcap asset row

Every mcap POST — whether through the API handler at ingest time or via retroactive backfill — MUST leave a raw_mcap asset row whose 7 columns mirror the source mcap-file's top-level (and one JSONB-blob-lifted) fields:

| Asset column | Source | Notes |
|---|---|---|
| `camera_model` | `mcap_files.camera_model` | text, e.g. `"CyberCap2"` |
| `device_id` | `mcap_files.device_id` (coerced to uuid) | nullable when producer sends empty |
| `collector_id` | `mcap_files.collector_id` (coerced to uuid) | |
| `scene_id` | `mcap_files.scene_id` (coerced to uuid) | |
| `data_source` | `mcap_files.data_source` | text, e.g. `"vendor"` |
| `collection_method` | `mcap_files.collection_method` | text, e.g. `"human"` |
| `source_platform` | `mcap_files.metadata ->> 'source_platform'` | lifted from JSONB to first-class column |

The columns are all nullable (some legacy or non-mcap-derived assets won't have them).

#### Scenario: mcap POST populates all 7 asset columns

- **Given** a `POST /api/v1/mcap-files` body with all 7 fields populated (`camera_model=CyberCap2`, `device_id=<uuid>`, `collector_id=<uuid>`, `scene_id=<uuid>`, `data_source=vendor`, `collection_method=human`, `metadata.source_platform=vibecap`)
- **When** the request succeeds
- **Then** the auto-derived raw_mcap asset row has all 7 columns populated with the same values (uuids coerced from text).

#### Scenario: post-migration backfill covers historical mcap

- **Given** N historical mcap-file rows and N corresponding raw_mcap asset rows created before this migration
- **When** the migration `<timestamp>_flatten_mcap_columns_to_assets.sql` runs
- **Then** every historical asset row has its 7 columns backfilled from the joined mcap-file row; rows whose mcap_file source has a null/empty field remain null in the asset column.

### Requirement: 7 new mcap columns are filterable + 4 are facetable via /queries/run

`POST /api/v1/queries/run` MUST accept the following field names in `where` predicates and (for a subset) as facet fields:

| Field | Filter | Facet |
|---|---|---|
| `camera_model` | ✓ | ✓ |
| `data_source` | ✓ | ✓ |
| `collection_method` | ✓ | ✓ |
| `source_platform` | ✓ | ✓ |
| `device_id` | ✓ | ✗ (too many buckets) |
| `collector_id` | ✓ | ✗ |
| `scene_id` | ✓ | ✗ |

The current 422 UNSUPPORTED_FIELD response for these fields disappears.

#### Scenario: filter by camera_model returns matches

- **Given** at least one asset with `camera_model="CyberCap2"`
- **When** the client sends `POST /queries/run` with `where: {pred: {field: "camera_model", op: "eq", value: "CyberCap2"}}`
- **Then** the response `total > 0` and every returned item has `camera_model="CyberCap2"`.

#### Scenario: facet on source_platform returns non-empty buckets

- **When** the client sends `POST /queries/run` with `facets: [{field: "source_platform"}]`
- **Then** `facets.source_platform` includes a bucket for `vibecap` (and any other producer platforms) with counts matching the number of assets carrying that value.

#### Scenario: device_id supports filter but is refused as facet field

- **When** the client sends `facets: [{field: "device_id"}]`
- **Then** the request returns 422 UNSUPPORTED_FIELD (facet only) — same field on `where` works.

## MODIFIED

### Requirement: raw_mcap asset creation propagates full mcap-file identity

Previously the auto-derived raw_mcap asset only mirrored `mcap_file_id / start_timestamp_ns / end_timestamp_ns / duration_ms / owner / retention_tier / expire_at / tenant / project`. It now additionally mirrors the 7 columns listed above. The rest of the placeholder shape (asset_type=raw_mcap, lifecycle_state=created, etc.) is unchanged.

Existing callers of `assetRepo.InsertNew` that don't populate these 7 fields still work — they simply produce a raw_mcap row with nullable columns unset. This preserves compatibility for tests + non-mcap paths.
