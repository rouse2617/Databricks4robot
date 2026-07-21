# Spec delta — tags (mcap ingest side-effect)

## ADDED

### Requirement: mcap ingest auto-writes task / source / city tags from metadata

`POST /api/v1/mcap-files` MUST, in the same transaction as the mcap and raw_mcap-asset writes, extract three specific fields from the mcap's `metadata` JSONB and insert corresponding tag rows via `AssetTagRepository.Upsert`. The rules are hardcoded:

| Metadata path | Tag key | Cardinality | Extraction |
|---|---|---|---|
| `metadata.vibecap_tasks` | `task` | multi (one row per value) | Array of strings, or JSON-encoded string that parses to array |
| `metadata.source_platform` | `source` | single | Direct string |
| `metadata.location.address` | `city` | single | Reverse-iterate comma-split segments; take first segment ending in `市` that is ≥3 chars AND does not end in `超市`, `大厦`, `商店`, `商场`, `广场`, `医院` |

Every tag uses `source_type="system"`, `source_name="mcap_ingest"`. Only mcap-specific known keys are extracted — every other key in `metadata` (`collector_height`, `collection_session_id`, `weather`, etc.) is preserved in the JSONB blob but is NOT auto-promoted to a tag.

#### Scenario: pangzi-shape mcap POST populates 3 tags

- **Given** a `POST /api/v1/mcap-files` body with `metadata: {vibecap_tasks: ["备餐操作","台面清洁"], source_platform: "vibecap", location: {address: "合肥新民医院, ..., 合肥市, 安徽省, 230061, 中国"}}`
- **When** the request succeeds
- **Then** the asset_tags table gains rows: `(task, 备餐操作)`, `(task, 台面清洁)`, `(source, vibecap)`, `(city, 合肥市)` — all with `source_type="system"`, `source_name="mcap_ingest"`, and the same `asset_id` as the auto-derived raw_mcap.

#### Scenario: value shape mismatch skips that tag without failing

- **Given** metadata `{vibecap_tasks: 42}` (number, not array or JSON-string)
- **When** the mcap POST is processed
- **Then** the mcap and raw_mcap asset are created successfully, no `task` tag row is inserted, and a WARN log line records `metadata tag extract skip: vibecap_tasks type unexpected`.

#### Scenario: unknown metadata keys are ignored

- **Given** metadata `{vibecap_tasks: ["A"], source_platform: "vibecap", weather: "sunny", collector_height: 1.6}`
- **When** processed
- **Then** exactly 2 tag rows are inserted (`task:A`, `source:vibecap`); no tag rows are written for `weather` or `collector_height`; both are still preserved verbatim in the mcap `metadata` JSONB.

#### Scenario: single-segment address skips city

- **Given** metadata `{location: {address: "惠乐超市"}}`
- **When** processed
- **Then** no `city` tag row is inserted (single segment cannot reliably yield a city name; POI-name endings are filtered out even if they end in `市`).

#### Scenario: address with POI-name in first segment still extracts city correctly

- **Given** `location.address = "惠乐超市, 合肥市, 安徽省, 中国"`
- **When** processed
- **Then** `city: 合肥市` is inserted (reverse iteration + POI filter + min length ≥3 skips `惠乐超市`, picks `合肥市`).

## MODIFIED

### Requirement: mcap-files POST is now a superset of tag-write side effects

Previously `POST /api/v1/mcap-files` inserted the mcap row, the auto-derived raw_mcap asset row, and outbox events. It now ALSO inserts up to N tag rows (where N is the number of extractable rules that matched the payload). Callers cannot opt out; the extraction is unconditional and runs inside the same transaction, so a tag-write failure will roll back the mcap create just like any other tx failure.

The response body remains unchanged (the mcap file struct + created_at / updated_at fields). Clients that read the mcap after create can call `GET /api/v1/assets/{mcap_file_id}` to observe the newly written tags.
