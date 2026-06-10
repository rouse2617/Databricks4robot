## ADDED Requirements

### Requirement: Multi-source tag coexistence
The system SHALL persist tag assertions from different sources on the same
`(asset_id, tag_key)` without overwriting each other.

**Priority**: P0 (Critical)
**Rationale**: Delivery rules, compliance, and UI need to see what each
source (human / algo / rule / compliance / system) asserts.

#### Scenario: Two sources, same key
- **Given** asset `a1` has no tags
- **When** `algo_sdk` upserts `scenario=kitchen` and `human` upserts
  `scenario=kitchen` on the same asset
- **Then** `GET /assets/a1` returns two entries in `tags_detailed` with
  `source_type` `algo_sdk` and `human` respectively

#### Scenario: Same source, idempotent re-write
- **Given** `algo_sdk` source_version `2.0` has already tagged
  `scenario=kitchen` on `a1`
- **When** the same source/version writes the same `(key, value)` again
- **Then** the row count for `(a1, scenario, kitchen, algo_sdk, 2.0)` is `1`

#### Scenario: Same source, conflicting values
- **Given** human `labeler_A` tagged `scenario=kitchen` on `a1`
- **When** human `labeler_B` tags `scenario=warehouse` on `a1`
- **Then** both rows persist; UI displays both with reviewer identity

### Requirement: Source-scoped tag delete
The system SHALL allow deleting a tag for a specific source without
removing assertions from other sources.

**Priority**: P0 (Critical)

#### Scenario: Delete one source
- **Given** `a1` has `scenario=kitchen` from `human` and `rule_engine`
- **When** client DELETEs `/assets/a1/tags/scenario?source_type=human`
- **Then** only the `human` row is removed; `rule_engine` row remains

#### Scenario: Delete all sources for key
- **Given** `a1` has multiple sources for key `scenario`
- **When** client DELETEs `/assets/a1/tags/scenario` with no source filter
- **Then** all `scenario` rows for `a1` are removed

### Requirement: Source identity captured on upsert
The system SHALL persist `source_name`, `source_version`, and (when
applicable) `run_id` on every tag upsert.

**Priority**: P0 (Critical)
**Rationale**: Today's repo drops these columns; design doc §3.3.

#### Scenario: Algo upsert with run
- **When** `POST /assets/a1/tags` body is
  `{key:'algo.hand_track.status', value:'ok', source_type:'algo_sdk',
   source_name:'hand_track', source_version:'2.0', run_id:'r_42'}`
- **Then** the persisted row has all four source columns populated

### Requirement: Registry enforces source contracts
The system SHALL validate tag writes against `tag_registry.yaml`
`tag_sources[]` rules and reject writes that violate
`writable_by` / `requires_source_name` / `requires_source_version` /
`immutable`.

**Priority**: P1 (High)

#### Scenario: Human requires source_name
- **Given** registry rule `source: human, requires_source_name: true`
- **When** client POSTs `{key:'scenario', value:'kitchen', source_type:'human'}`
  without `source_name`
- **Then** response is `422` with `code=tag_source_invalid`

#### Scenario: Compliance immutable
- **Given** `a1` already has `compliance.pii=true` from source `compliance`
- **When** any caller attempts to update the same row
- **Then** response is `409` with `code=tag_immutable`

### Requirement: Detailed tags in asset response
The system SHALL return both `tags` (flat map, back-compat) and
`tags_detailed` (array with full source identity) on
`GET /assets/{id}`.

**Priority**: P1 (High)

#### Scenario: Asset response shape
- **When** client GETs an asset with multi-source tags
- **Then** response includes `tags_detailed: [{key, value, source_type,
  source_name, source_version, run_id, applied_at}]` and `tags` map where
  each key maps to the value of the row with greatest `applied_at`
