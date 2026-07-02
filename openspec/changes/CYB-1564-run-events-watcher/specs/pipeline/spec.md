## ADDED Requirements

### Requirement: Durable pipeline run event timeline
The system SHALL persist a chronological event timeline for each pipeline run that includes run actions, workflow observations, node transitions, Pod observations, and terminal outcomes.

**Priority**: P0 (Critical)
**Rationale**: Users need to understand what happened during a run even after Argo workflows or Pods are cleaned up.

#### Scenario: Submitted run appears in timeline
- **Given** a user starts a pipeline run
- **When** the run is accepted by DataBrew
- **Then** the run detail timeline shows a `run_submitted` event with the run ID, workflow name when known, and submission time

#### Scenario: Completed workflow remains explainable
- **Given** a pipeline run has completed successfully and its workflow is later cleaned up by Argo retention
- **When** the user opens the run detail page
- **Then** DataBrew still shows the stored run, workflow, and node completion events from its own event ledger

#### Scenario: Failed node records failure context
- **Given** Argo reports that a workflow node failed with a message or reason
- **When** the watcher observes the failed node
- **Then** DataBrew records a node failure event containing the node identity, status, and best available diagnostic message

### Requirement: Idempotent Argo event synchronization
The system SHALL synchronize active pipeline runs from Argo into the run event ledger without duplicating the same observed transition across watcher ticks or backend restarts.

**Priority**: P0 (Critical)
**Rationale**: Polling is practical for P0.1, but duplicate observations would make timelines noisy and undermine audit quality.

#### Scenario: Repeated polling does not duplicate a node transition
- **Given** the watcher has already recorded that node `step-a` succeeded
- **When** the watcher observes the same workflow node success again
- **Then** no duplicate `node_succeeded` event is added for the same run and node transition

#### Scenario: Watcher restart preserves deduplication
- **Given** the backend restarts after recording workflow status changes
- **When** the watcher resumes and observes the same active workflow state
- **Then** persisted idempotency keys prevent duplicate events

### Requirement: Pipeline run events API
The system SHALL expose a paginated API for listing stored events for a pipeline run.

**Priority**: P0 (Critical)
**Rationale**: Frontend, SDK, smoke tests, and future operators need a stable DataBrew API instead of direct Argo/Kubernetes access.

#### Scenario: List events for an existing run
- **Given** a pipeline run has recorded events
- **When** a client requests the run events API for that run
- **Then** the API returns chronological events with pagination metadata

#### Scenario: Reject invalid pagination
- **Given** a client provides an invalid cursor or limit
- **When** the client requests the run events API
- **Then** the API returns an invalid-argument error and does not query Argo directly

#### Scenario: Unknown run
- **Given** no pipeline run exists for the requested ID
- **When** a client requests the run events API
- **Then** the API returns a not-found error

## MODIFIED Requirements

### Requirement: Run execution record
- **Before**: The system SHALL show pipeline runs as execution records with pipeline name, asset count, execution target, workflow name, status, timestamps, and available actions.
- **After**: The system SHALL show execution records and details with a durable run event timeline that explains submission, scheduling, node, Pod, retry, and terminal changes.
- **Reason**: Status alone does not explain why a run failed, retried, or changed state.

#### Scenario: Detail page shows real run events
- **Given** a pipeline run has stored events
- **When** the user opens the run detail page
- **Then** the "运行事件" section displays those events instead of a backend-placeholder message
