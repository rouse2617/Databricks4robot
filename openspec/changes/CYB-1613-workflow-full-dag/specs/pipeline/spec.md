# Pipeline Spec Delta — CYB-1613

## MODIFIED Requirements

### Requirement: Workflow detail returns complete DAG topology
- **Before**: The system SHALL return workflow detail nodes based on Argo runtime `status.nodes`.
- **After**: The system SHALL return workflow detail nodes by combining Argo runtime `status.nodes` with static DAG task definitions from `spec.templates`, so not-yet-started DAG tasks are visible as pending nodes.
- **Reason**: Argo does not create all downstream `status.nodes` when a workflow starts; rendering only runtime nodes makes the UI hide future steps.

**Priority**: P0 (Critical)
**Rationale**: Users need to understand the full workflow shape immediately, especially while an early step is running or failed.

#### Scenario: Running workflow shows future tasks
- **Given** a workflow spec defines tasks `step-prepare`, `step-checksum`, `step-validate`, and `step-store`
- **And** Argo status currently contains only `step-prepare` and `step-checksum`
- **When** the client requests workflow detail
- **Then** the response includes all four tasks in `nodes`
- **And** the two missing tasks have pending/not-yet-started state.

#### Scenario: Runtime nodes keep full metadata
- **Given** a task exists in Argo runtime status with pod, phase, timing, outputs, and debug capability metadata
- **When** the handler merges runtime and static DAG data
- **Then** the runtime node fields are preserved and are not replaced by the static task placeholder.

#### Scenario: Static dependency edges target pending tasks
- **Given** a DAG task dependency points to a task that is not yet present in runtime status
- **When** the handler builds workflow edges
- **Then** the response includes a static DAG edge to the pending task node.

#### Scenario: Non-DAG templates are not shown as pending workflow steps
- **Given** the workflow spec contains container templates, exit handlers, or helper templates that are not DAG tasks
- **When** the handler builds static pending nodes
- **Then** those helper templates are not returned as independent pending DAG nodes.
