# Pipeline Spec Delta — CYB-1534 Run Target Model

## New Requirements

### Requirement: Persist execution targets
The system SHALL persist execution targets as configurable runtime destinations for pipeline runs.

#### Scenario: List persisted execution targets
- **Given** at least one execution target exists
- **When** a client lists execution targets
- **Then** the response returns target metadata without exposing raw secrets.

#### Scenario: Default target compatibility
- **Given** a client omits `target_id` or sends `target_id=default`
- **When** the client creates a pipeline run
- **Then** the system uses the enabled default execution target.

### Requirement: Persist pipeline runs
The system SHALL persist pipeline run records independently from legacy deployment records.

#### Scenario: Create run from template
- **Given** a saved pipeline template exists
- **When** a client creates a pipeline run with that template ID
- **Then** the system persists a run with pipeline snapshot, asset IDs, execution target ID, workflow name, and status.

#### Scenario: Run survives Argo TTL cleanup
- **Given** a run row exists and the Argo Workflow CR has been cleaned up
- **When** the system refreshes the run status
- **Then** the persisted run is marked `Expired` rather than disappearing.

### Requirement: Persist pipeline run nodes
The system SHALL persist node-level run records for pipeline execution steps.

#### Scenario: Refresh run nodes from Argo
- **Given** an active run has an Argo workflow
- **When** the system refreshes run detail
- **Then** it upserts run node records with Argo node ID, pod name, phase, message, timing, and resource duration metadata.

### Requirement: Preserve deployment compatibility
The system SHALL preserve legacy deployment endpoints while first-class pipeline runs are introduced.

#### Scenario: Existing deployment clients still work
- **Given** a client calls an existing deployment endpoint
- **When** the request succeeds
- **Then** the response shape remains compatible with existing `PipelineDeployment` clients.
