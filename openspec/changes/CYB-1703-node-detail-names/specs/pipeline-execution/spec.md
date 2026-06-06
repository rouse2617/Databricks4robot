## MODIFIED Requirements

### Requirement: Workflow node detail table uses business node labels
- **Before**: The workflow execution detail node table could display Argo technical step ids such as `step-step-2` even when the DAG cards displayed the pipeline component name.
- **After**: The workflow execution detail node table SHALL display the same business/component label used by the DAG card when that label is available, while preserving technical ids for diagnostics.
- **Reason**: Operators identify pipeline steps by component labels. Showing only technical Argo ids in the table makes status, cost, logs, and Pod diagnostics harder to correlate with the DAG.

**Priority**: P1 (High)
**Rationale**: The node detail table is the primary place users inspect per-step status and logs after opening an execution. Its labels must match the visual DAG.

#### Scenario: node table shows component name
- **Given** a workflow execution has two Argo step nodes `step-step-2` and `step-step-3`
- **And** the DAG cards display the component label `Count Lines`
- **When** the user opens the workflow execution detail page
- **Then** the node detail table displays `Count Lines` for those rows instead of only `step-step-2` or `step-step-3`

#### Scenario: technical id remains available
- **Given** a node table row is displayed with the component label `Count Lines`
- **When** the user needs to diagnose logs or Pod state
- **Then** the technical node id remains available as secondary information or tooltip

#### Scenario: missing business label falls back safely
- **Given** a workflow node does not have a component or business display label
- **When** the user opens the node detail table
- **Then** the table falls back to the existing technical node label without hiding status, logs, Pod diagnostics, or cost information
