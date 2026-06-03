## ADDED Requirements

### Requirement: Durable run event ledger
The system SHALL store a durable timeline for pipeline run, workflow, pod, node, retry, resubmit, and delete lifecycle events.

**Priority**: P0 (Critical)
**Rationale**: Users need execution history after Argo cleanup and need to locate failures by run, node, pod, and asset.

#### Scenario: Workflow lifecycle events are stored
- **Given** a pipeline run creates an Argo workflow
- **When** the watcher observes the workflow
- **Then** DataBrew stores workflow creation/observation and phase-change events with idempotency keys

#### Scenario: Node and pod lifecycle events are stored
- **Given** an Argo workflow has pod-backed nodes
- **When** the watcher observes node start, terminal phase, or pod creation
- **Then** DataBrew stores node and pod events with node/pod identifiers and status

#### Scenario: User operations are audited
- **Given** a user retries, resubmits, or deletes a pipeline run
- **When** the operation is requested
- **Then** DataBrew stores a run event for the request before or alongside downstream side effects

### Requirement: DataBrew history survives Argo cleanup
The system SHALL render stored run, node, asset-node, and event history even when the Argo workflow is expired or unavailable.

**Priority**: P0 (Critical)
**Rationale**: Argo workflow TTL must not delete product-visible execution history.

#### Scenario: Argo workflow is expired
- **Given** DataBrew has stored run history and Argo no longer returns the workflow
- **When** the user opens the run detail page
- **Then** the UI shows DataBrew stored timeline and node snapshots with a clear "Argo expired" state

#### Scenario: No DataBrew history exists
- **Given** neither Argo nor DataBrew has usable history for a workflow
- **When** the user opens the detail page
- **Then** the UI shows a clear empty/error state instead of a blank DAG or raw backend placeholder

### Requirement: Watcher health visibility
The system SHALL expose persistent watcher health state for pipeline run synchronization.

**Priority**: P1 (High)
**Rationale**: If the ledger is DataBrew's source of truth, users and operators need to know whether synchronization is healthy or stale.

#### Scenario: Watcher sync succeeds
- **Given** the watcher scans active runs successfully
- **When** watcher status is requested
- **Then** the system returns last scan time, last success time, synced count, active scan limit, and zero consecutive failures

#### Scenario: Watcher sync fails
- **Given** the watcher cannot list or refresh runs
- **When** watcher status is requested
- **Then** the system returns last error, last error time, consecutive failure count, total error count, and stale status

### Requirement: Event timeline filtering and pagination
The system SHALL provide filtered and paginated access to run events.

**Priority**: P1 (High)
**Rationale**: Large or long-running pipelines may have many events, and the UI must stay usable.

#### Scenario: User filters event timeline
- **Given** a run has mixed event types and statuses
- **When** the user filters by event type, subject type, status, search text, or time range
- **Then** the API returns matching events with total and next cursor metadata

#### Scenario: User loads more events
- **Given** a run has more events than the first page limit
- **When** the user requests the next cursor
- **Then** the API returns the next chronological page without duplicating prior events
