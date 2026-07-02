## MODIFIED Requirements

### Requirement: Pipeline workflow logs are inspectable from node detail
- **Before**: The system exposed workflow node logs through the workflow detail UI, but Pod nodes could return empty logs because the Argo node id was used as the Kubernetes pod name.
- **After**: The system SHALL resolve the Kubernetes pod backing a selected Argo Pod node and show its logs in the workflow detail UI.
- **Reason**: Operators need logs to debug succeeded and failed algorithm steps; empty logs hide real execution evidence.

**Priority**: P0 (Critical)
**Rationale**: A run control plane is not usable for algorithm debugging if users cannot inspect pod logs from the UI.

#### Scenario: Succeeded pod node shows logs
- **Given** a workflow Pod node has completed and its Kubernetes pod has logs
- **When** the user opens the node log view in workflow detail
- **Then** the UI shows the pod logs for that node
- **And** the log request does not require the user to know the Kubernetes pod name

#### Scenario: Missing pod logs show a clear error
- **Given** a workflow Pod node cannot be resolved to a Kubernetes pod
- **When** the user opens the node log view
- **Then** the UI shows a clear failure state instead of a blank log panel

### Requirement: Pipeline designer drops the selected component
- **Before**: Dragging a component from the palette could create a node for a different component.
- **After**: The system SHALL create a canvas node from the exact palette component selected by the user's drag action.
- **Reason**: Saving and running the wrong component can produce incorrect pipeline behavior and invalid execution results.

**Priority**: P0 (Critical)
**Rationale**: Component identity is the core contract of a visual pipeline designer.

#### Scenario: Dragging a specific component creates that component
- **Given** the component palette contains `codex-valid-emit-*` and `codex-valid-transform-*`
- **When** the user drags `codex-valid-emit-*` onto the canvas
- **Then** the created node is named `codex-valid-emit-*`
- **And** its image, command, args, and resources come from the emit component

#### Scenario: Repeated drags preserve identity
- **Given** the user has cleared the canvas
- **When** the user drags multiple different components onto the canvas
- **Then** each created node matches the component that was dragged
