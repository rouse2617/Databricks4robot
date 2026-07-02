## ADDED Requirements

### Requirement: Component resources support GPU and compute tier
The system SHALL allow pipeline components and pipeline node resources to include `gpu` and `computeTier` values in addition to CPU, memory, and disk.

#### Scenario: Component resource fields round trip
- **Given** a user creates or edits a pipeline component
- **When** they set GPU and compute tier resource fields
- **Then** the component API response SHALL include those values under `resources`
- **And** dragging the component into the pipeline designer SHALL preserve those values on the node.

### Requirement: GPU resources are emitted to Argo manifests
The system SHALL map `resources.gpu` to the Kubernetes extended resource `nvidia.com/gpu` when generating Argo workflow templates.

#### Scenario: GPU component deploys as a GPU pod
- **Given** a pipeline node declares `resources.gpu` as `"1"`
- **When** DataBrew generates the Argo workflow manifest
- **Then** the node template SHALL include `limits["nvidia.com/gpu"] == "1"`.
