## ADDED Requirements

### Requirement: Asset-node execution matrix
The system SHALL expose asset × pipeline-node execution rows for each pipeline run, including status, timestamps, log/pod/resource affordances, and estimated cost source.

**Priority**: P0 (Critical)
**Rationale**: Users need to debug which asset is affected by which pipeline step instead of reading only run-level status.

#### Scenario: Asset-backed run shows asset-node rows
- **Given** a pipeline run has selected assets and node snapshots
- **When** the user opens the run detail page
- **Then** the system shows rows for each selected asset and pipeline node with status and inspection actions

#### Scenario: No-asset run still shows node rows
- **Given** a no-asset pipeline run has node snapshots
- **When** the user opens the asset-node section
- **Then** the system shows a no-asset row for each node instead of an empty section

### Requirement: Estimated cost and audit summary
The system SHALL expose estimated run/node/asset-node cost summaries with source labels.

**Priority**: P0 (Critical)
**Rationale**: Operators need to identify slow and expensive steps before exact billing integration exists.

#### Scenario: Cost summary uses node estimated costs
- **Given** a run has nodes with estimated costs
- **When** the user requests the cost summary
- **Then** the system returns total estimated cost and per-node summary with `estimated` source labels

#### Scenario: Cost unavailable is explicit
- **Given** a run has no resource duration or pricing information
- **When** the user opens the cost summary
- **Then** the UI shows cost unavailable rather than zero-cost

### Requirement: Run event timeline filtering
The system SHALL let users filter and search stored run events by type, status, subject, text query, and time range.

**Priority**: P1 (High)
**Rationale**: Timelines become hard to scan once runs have many node and Pod events.

#### Scenario: Filter failed node events
- **Given** a run has mixed success and failure events
- **When** the user filters event type or status to failures
- **Then** only matching failure events remain visible

#### Scenario: Load more timeline events
- **Given** a run has more events than the initial page
- **When** the user selects load more
- **Then** the next chronological page is appended without losing current filters

### Requirement: Notification candidate creation
The system SHALL create idempotent notification candidates for failed or error run events.

**Priority**: P1 (High)
**Rationale**: External notifications require exactly-once event selection before provider delivery is added.

#### Scenario: Failed event creates one notification candidate
- **Given** a node failure event is recorded
- **When** notification candidate generation runs
- **Then** exactly one candidate is created for that event

#### Scenario: Repeated watcher sync does not duplicate notification candidates
- **Given** a failure event already has a notification candidate
- **When** the watcher observes the same failure again
- **Then** no duplicate notification candidate is created

### Requirement: Bounded watcher synchronization
The system SHALL persist watcher synchronization state and bound active-run scanning per tick.

**Priority**: P1 (High)
**Rationale**: Hundreds of workflows should not force the backend to rescan all historical runs on every tick.

#### Scenario: Watcher resumes after restart
- **Given** watcher state was persisted before backend restart
- **When** the backend starts again
- **Then** the watcher resumes bounded active-run synchronization using stored state
