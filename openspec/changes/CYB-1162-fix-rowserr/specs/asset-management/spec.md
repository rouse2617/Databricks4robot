## MODIFIED Requirements

### Requirement: Asset lineage precision and error transparency
The system SHALL surface partial or failed query results in the asset lineage
response with at least diagnostic logging when database iteration errors occur.

- **Before**: The system SHALL silently skip un-scannable rows in lineage queries
  and never check `rows.Err()` after iteration.
- **After**: The system SHALL check `rows.Err()` after each `rows.Next()` loop in
  lineage queries, and SHALL log scan errors at `slog.Warn` level for
  observability.
- **Reason**: Un-diagnosed iteration errors can silently truncate downstream
  lists (deliveries, algo results, eval results) displayed in the asset detail
  page, misleading users about available data.

**Priority**: P1 (High)
**Rationale**: While not crash-level, silent data loss in lineage display erodes
trust in the catalog and matches a known root cause (see commit 7f0cf5e).

#### Scenario: Three lineage queries all succeed
- **Given** all three SQL queries (`asset_algo_latest`, `delivery_items`,
  `asset_eval_results`) return valid rows and no iteration error
- **When** `buildLineageResponse` iterates each result set
- **Then** all rows are returned in the response with no diagnostics logged

#### Scenario: Database cursor error during algo results iteration
- **Given** the `asset_algo_latest` query returns 1 valid row then `rows.Err()`
  returns a connection error
- **When** `buildLineageResponse` completes its `rows.Next()` loop
- **Then** the already-scanned algo result row is included in the response,
  and the `rows.Err()` error is logged at `slog.Warn` (or higher)

#### Scenario: Postgres client not configured (no queries run)
- **Given** `h.pg` is nil when `buildLineageResponse` is called
- **When** the function executes
- **Then** an empty lineage skeleton is returned immediately with no queries attempted
