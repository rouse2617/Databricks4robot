## MODIFIED Requirements

### Requirement: Batch fan-out creates one logical subtask per asset
- **Before**: The system could create duplicate logical backfill items and duplicate visible child execution records for a single batch asset during background scheduling.
- **After**: The system SHALL create and track exactly one current logical backfill item and one default visible child execution record per submitted asset for a batch.
- **Reason**: Operators use batch totals to judge whether all assets were scheduled and processed; duplicate logical rows make a 100-asset batch look like 200 records.

**Priority**: P0 (Critical)
**Rationale**: Incorrect logical counts directly break batch progress, pagination, and failure triage.

#### Scenario: 100 submitted assets produce 100 logical children
- **Given** a user submits 100 unique asset IDs to run a published pipeline template as a batch
- **When** the backend creates and starts the batch
- **Then** the batch stores 100 logical items and the default child execution list reports a total of 100 records

#### Scenario: Scheduling reconciliation does not duplicate children
- **Given** a batch already has persisted logical items for its submitted assets
- **When** background scheduling or list reconciliation runs
- **Then** the system reuses those items and does not create a second logical item for the same asset

### Requirement: Batch completion status converges with logical item progress
- **Before**: A batch could report all logical items completed while the batch job status remained `running`.
- **After**: The system SHALL mark a non-paused batch as completed when all logical items have completed and no logical item failed.
- **Reason**: A running status after all work completed blocks reliable operator triage and follow-up automation.

**Priority**: P0 (Critical)
**Rationale**: Batch status is the primary signal for whether operators can stop monitoring a fan-out run.

#### Scenario: All items succeeded
- **Given** a batch has 100 logical items
- **When** all 100 logical items are completed and none failed
- **Then** the batch status is `completed`, completed count is 100, and failed count is 0

#### Scenario: Any item failed
- **Given** a batch has 100 logical items
- **When** at least one logical item failed and no item remains pending or running
- **Then** the batch status is `failed` and the failed count reflects the failed logical items

### Requirement: Batch node summary uses logical batch totals
- **Before**: Node summary coverage and subtask pending counts could include duplicate attempts or stale run rows, producing totals larger than the batch size.
- **After**: The system SHALL report node summary subtasks and data coverage against the current logical batch items, not all historical run attempts, unless an explicit attempts view is requested.
- **Reason**: Node-level success, pending, and failure rates must match the operator-facing batch size.

**Priority**: P0 (Critical)
**Rationale**: Node summary is used to diagnose which pipeline step is blocking or failing during large fan-out runs.

#### Scenario: Node rows cover every logical item
- **Given** a 100-item batch has node rows for each current logical item
- **When** the node summary is requested
- **Then** data coverage is complete, node attempted count is 100, and pending node count is 0

#### Scenario: Partial node row coverage
- **Given** a 100-item batch has node rows for only 60 current logical items
- **When** the node summary is requested
- **Then** data coverage is incomplete and node pending count reflects the 40 logical items without node rows
