## ADDED Requirements

### Requirement: Unschedulable pipeline runs converge to terminal error
The system SHALL mark a pipeline run as `Error` when Kubernetes scheduler diagnostics show that a workflow node has remained `Pending/Unschedulable` beyond the configured threshold.

**Priority**: P0 (Critical)
**Rationale**: Users cannot distinguish real work from impossible scheduling when a batch stays `running` forever after Kubernetes has already reported resource or taint constraints.

#### Scenario: Pending node becomes unschedulable error after threshold
- **Given** a pipeline run has a node in `Pending`
- **And** the node message includes Kubernetes scheduler diagnostics such as `Unschedulable`, `Insufficient cpu`, `Insufficient memory`, `Insufficient ephemeral-storage`, or `untolerated taint`
- **And** the node has remained pending longer than the configured unschedulable threshold
- **When** the watcher refreshes the run
- **Then** the run is marked `Error`
- **And** the scheduler diagnostic message remains visible on the run or node detail

#### Scenario: Short pending window remains non-terminal
- **Given** a pipeline run has a node in `Pending`
- **And** the node message includes a scheduler diagnostic
- **And** the pending duration is below the configured unschedulable threshold
- **When** the watcher refreshes the run
- **Then** the run remains non-terminal
- **And** the diagnostic message is still persisted for debugging

#### Scenario: Batch with unschedulable children stops running
- **Given** a batch execution has child pipeline runs that become `Error` due to unschedulable nodes
- **When** batch aggregation refreshes item and job status
- **Then** those child runs count as failed items
- **And** the parent batch no longer remains `running` after all children are terminal

### Requirement: Pipeline deployment validates configured resource ceilings
The system SHALL reject pipeline deployment before workflow submission when a component resource request exceeds configured execution-environment ceilings.

**Priority**: P0 (Critical)
**Rationale**: Users should receive an immediate, actionable validation error for obviously unsupported resource requests instead of creating workflows that cannot schedule.

#### Scenario: Component exceeds configured CPU or memory ceiling
- **Given** the execution environment is configured with maximum CPU and memory ceilings
- **And** a pipeline component requests CPU or memory above those ceilings
- **When** the user deploys the pipeline or creates a run
- **Then** the request fails before Argo workflow submission
- **And** the error identifies the component, requested resource, configured limit, and environment

#### Scenario: Unconfigured ceiling does not block deployment
- **Given** no static ceiling is configured for a resource type
- **When** a pipeline component requests that resource
- **Then** deployment validation does not reject the request for that resource type
- **And** runtime watcher safeguards still apply if Kubernetes later reports unschedulable diagnostics

#### Scenario: Valid resource requests continue to submit
- **Given** all pipeline component resource requests are within configured ceilings
- **When** the user deploys the pipeline or creates a run
- **Then** the workflow submission proceeds using the existing Argo generation path
