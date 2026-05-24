## ADDED Requirements

### Requirement: Silver asset-events export path
The system SHALL provide a repeatable Silver asset-events export path derived from Bronze asset events for downstream audit analytics.

**Priority**: P2 (Nice-to-have)
**Rationale**: Bronze ingestion may contain expected duplicates after retry, while audit analytics need stable Silver semantics.

#### Scenario: Silver export materializes current asset event state
- **Given** Bronze asset events exist in the lakehouse
- **When** the Silver asset-events export runs
- **Then** the lakehouse contains a Silver asset-events table with deduped/current-state rows suitable for analytical reads.

#### Scenario: Silver export can be rerun
- **Given** a previous Silver export has completed
- **When** the export is run again over the same Bronze input
- **Then** it produces deterministic Silver output without modifying Bronze rows.

### Requirement: Silver table visibility
The system SHALL expose Silver asset-events table availability through the existing lakehouse table listing behavior.

**Priority**: P2 (Nice-to-have)
**Rationale**: Operators need to verify whether Silver materialization exists without querying BigQuery manually.

#### Scenario: Silver table exists
- **Given** the lakehouse query layer can read `silver_asset_events_current`
- **When** a client requests lakehouse table metadata
- **Then** the response includes the Silver table name and row count.

#### Scenario: Lakehouse unavailable or Silver table missing
- **Given** the lakehouse query layer is disabled or the Silver table has not been created
- **When** smoke verification checks table visibility
- **Then** verification reports a tolerated missing/disabled state instead of treating it as a product API failure.
