## MODIFIED Requirements

### Requirement: Pipeline template detail accepts stable user-facing ids
- **Before**: The pipeline designer could receive a short template id from a user-facing link, but template detail lookup only resolved the full template UUID and left the designer in empty new-pipeline mode.
- **After**: The system SHALL resolve a short template id when it uniquely matches the prefix of one saved template UUID, and SHALL keep full UUID lookup behavior unchanged.
- **Reason**: Operators use displayed short ids in regression logs and links. A short id that already resolves versions must not open an empty designer when the saved template exists.

**Priority**: P1 (High)
**Rationale**: Broken saved-pipeline links make users think the template was lost and block pipeline edit/run workflows.

#### Scenario: unique short id rehydrates saved template
- **Given** a saved pipeline template has id `c00f1136-1f86-4a32-a25d-91bf81557fc8`
- **When** the user opens `/pipeline?templateId=c00f1136`
- **Then** the designer loads that saved template and shows its nodes and metadata

#### Scenario: full id remains supported
- **Given** a saved pipeline template has a full UUID id
- **When** the user opens `/pipeline?templateId=<full-uuid>`
- **Then** the designer loads the same saved template as before

#### Scenario: missing or ambiguous short id is not guessed
- **Given** no saved template uniquely matches a short id prefix
- **When** the user opens the short-id link
- **Then** the system treats it as not found instead of loading an unrelated template

### Requirement: Pipeline run asset selection can use exact asset ids
- **Before**: The pipeline run modal only used search results, so an existing asset id such as `CYB10A01` could be unavailable when the search index returned zero rows.
- **After**: The system SHALL attempt a direct asset lookup for exact asset id queries when search returns no matches, and SHALL offer the asset for selection when the lookup succeeds.
- **Reason**: Asset ids are canonical identifiers. A search index miss must not prevent running a pipeline against a known existing asset.

**Priority**: P1 (High)
**Rationale**: The run modal is a primary execution entry point; inability to select a known asset blocks end-to-end pipeline testing.

#### Scenario: direct asset id fallback succeeds
- **Given** `/search/assets?q=CYB10A01` returns zero rows
- **And** `/assets/CYB10A01` returns an existing asset
- **When** the user searches `CYB10A01` in the run modal
- **Then** the modal displays `CYB10A01` as a selectable result

#### Scenario: direct lookup miss keeps empty state
- **Given** search returns zero rows
- **And** direct asset lookup also does not find the query
- **When** the user searches that query in the run modal
- **Then** the modal keeps the existing no-results empty state
