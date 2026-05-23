## ADDED Requirements

### Requirement: Logical asset ratings history
The system SHALL provide a read-only ratings history for a logical asset across its non-deleted revisions.

**Priority**: P1 (High)
**Rationale**: Operators need to compare quality/rating signals across versions without manually opening each revision.

#### Scenario: Fetch cross-revision ratings history
- **Given** a logical asset has multiple non-deleted revisions
- **And** one or more revisions have algorithm rating rows
- **When** a client requests `GET /api/v1/logical-assets/{id}/ratings-history`
- **Then** the response SHALL include `logical_asset_id`, `items`, and `count`
- **And** `items` SHALL contain one entry per non-deleted revision
- **And** each revision entry SHALL include its rating rows under `ratings`

#### Scenario: Revision with no ratings
- **Given** a logical asset revision exists
- **And** that revision has no rating rows
- **When** a client requests ratings history for the logical asset
- **Then** the response SHALL include that revision with `ratings: []`

#### Scenario: Missing logical asset
- **Given** no non-deleted revision rows exist for the requested logical asset id
- **When** a client requests ratings history
- **Then** the system SHALL return `404 ASSET_NOT_FOUND`

#### Scenario: Invalid logical asset id
- **Given** a client provides a blank or malformed logical asset id
- **When** the client requests ratings history
- **Then** the system SHALL return `400 INVALID_ARGUMENT`

### Requirement: Ratings history ordering
The system SHALL return ratings history in deterministic revision and rating order.

**Priority**: P1 (High)
**Rationale**: Trend charts and regression review need stable ordering independent of database row order.

#### Scenario: Revision order
- **Given** a logical asset has revisions created over time
- **When** a client requests ratings history
- **Then** revision items SHALL be ordered by `revision ASC`
- **And** `created_at ASC` SHALL break ties for legacy rows

#### Scenario: Rating row order
- **Given** one revision has multiple rating rows
- **When** a client requests ratings history
- **Then** ratings within the revision SHALL be ordered by `metric_key ASC`
- **And** `source_type ASC`, `source_name ASC`, and `recorded_at ASC` SHALL break ties

### Requirement: Ratings history source semantics
The system SHALL expose rating metric rows from the per-asset metric projection for CYB-1100.

**Priority**: P1 (High)
**Rationale**: Product docs define `asset_metrics` rows with `metric_key` in the `rating.*` namespace as the quality/rating signals that are compared across logical asset revisions.

#### Scenario: Rating metric fields
- **Given** an `asset_metrics` row exists for a revision with a `rating.*` metric key
- **When** ratings history is fetched
- **Then** the rating row SHALL include metric identity, typed metric values, source identity, eval identity, target identity, run id, confidence, and recorded time when present

#### Scenario: Non-rating metrics remain separate
- **Given** an asset has metric rows whose keys do not start with `rating.`
- **When** ratings history is fetched in CYB-1100
- **Then** the system SHALL NOT include those non-rating metric rows in the ratings response

#### Scenario: Algorithm state remains separate
- **Given** an asset has `asset_algo_latest` rows with result tags or scores
- **When** ratings history is fetched in CYB-1100
- **Then** the system SHALL NOT treat algorithm state rows as rating history unless a later contract explicitly adds that source type
