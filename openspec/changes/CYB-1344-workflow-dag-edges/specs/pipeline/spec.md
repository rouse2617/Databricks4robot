## ADDED Requirements

### Requirement: Workflow detail exposes normalized DAG edges
The system SHALL include normalized workflow DAG edges in the workflow detail response so clients can render dependency lines without inferring topology from partial runtime child links.

**Priority**: P0 (Critical)
**Rationale**: Runtime child links can omit logical dependencies when downstream steps are `Omitted` or `Skipped`, causing the workflow detail DAG to hide the relationship between failed upstream nodes and omitted downstream nodes.

#### Scenario: Failed upstream step connects to omitted downstream step
- **Given** a two-step workflow where the first step failed and the second step is omitted because its dependency was not satisfied
- **When** the workflow detail API returns the workflow nodes
- **Then** the response includes an edge whose source is the failed step and whose target is the omitted step

#### Scenario: Unresolvable graph endpoints are not emitted
- **Given** a workflow contains internal root or grouping nodes that the workflow detail DAG does not display
- **When** the backend normalizes workflow DAG edges
- **Then** emitted edges reference only nodes that can be rendered or are omitted from the normalized edge list

### Requirement: Workflow detail DAG renders backend-provided edges
The system SHALL render backend-provided workflow DAG edges before falling back to local edge inference.

**Priority**: P0 (Critical)
**Rationale**: The backend-normalized edge contract is the source of truth for workflow topology and avoids duplicated Argo-specific graph logic in frontend views.

#### Scenario: Backend edge is rendered between visible nodes
- **Given** the workflow detail response includes visible nodes and a normalized edge between them
- **When** the user opens the workflow detail DAG
- **Then** the DAG draws a connection line between the source and target nodes

#### Scenario: Missing edge field remains backward compatible
- **Given** the workflow detail response has nodes but no `edges` field
- **When** the user opens the workflow detail DAG
- **Then** the frontend uses its existing local fallback inference instead of failing to render the DAG
