## ADDED Requirements

### Requirement: Components may define default tolerations

Pipeline components MUST be able to store one or more default tolerations as part of their authored scheduling metadata.

#### Scenario: Component default tolerations are preserved

- **Given** a component author saves a component with toleration definitions
- **When** DataBrew persists and later reloads that component
- **Then** the toleration list is preserved in the component payload

### Requirement: Nodes may override tolerations

Authored pipeline nodes MUST be able to carry their own tolerations independent of the component registry record.

#### Scenario: Node-level tolerations are saved

- **Given** a user edits a node in the pipeline editor
- **When** they add or update tolerations in node configuration
- **Then** the saved node data contains those tolerations

### Requirement: New nodes inherit component tolerations

Newly created nodes from a registered component MUST inherit the component's default tolerations at creation time.

#### Scenario: Dragging a component copies default tolerations

- **Given** a component has default tolerations configured
- **When** the user drags that component into a pipeline canvas
- **Then** the created node includes the same tolerations in node data

### Requirement: Argo templates include authored tolerations

The transpiler MUST emit node tolerations into generated Argo templates for both container and script modes.

#### Scenario: Container template includes tolerations

- **Given** a container-mode node has tolerations
- **When** DataBrew transpiles the pipeline into an Argo workflow
- **Then** the generated template includes those tolerations

#### Scenario: Script template includes tolerations

- **Given** a script-mode node has tolerations
- **When** DataBrew transpiles the pipeline into an Argo workflow
- **Then** the generated script template includes those tolerations

### Requirement: GPU tier preset must not encode environment-specific scheduling

Compute-tier presets MAY add generic GPU tolerations, but MUST NOT hardcode environment-specific tolerations.

#### Scenario: gpu-l4 preset injects only GPU toleration

- **Given** a component uses `computeTier=gpu-l4`
- **When** DataBrew applies default scheduling presets
- **Then** the component receives the GPU toleration preset
- **And** it does not implicitly receive `environment=dev` or other environment-specific tolerations
