# Pipeline Spec Delta — CYB-1619 Complex Pipeline Authoring

## ADDED Requirements

### Requirement: Component ports are editable from the UI

The system SHALL let users create, edit, and remove component input and output ports from the component management UI without editing JSON.

**Priority**: P0
**Rationale**: Complex pipelines require explicit input and output ports, especially for fan-in joins and output contracts.

#### Scenario: User adds an output port

- **Given** a user is creating or editing a pipeline component
- **When** the user adds an output port named `summary`
- **Then** the component is saved with `summary` as an output port
- **And** the pipeline designer can use `component.summary` as a source output

#### Scenario: Output contract is visible

- **Given** a component has an output port named `summary`
- **When** the user edits the component command or script
- **Then** the UI explains that consumed output `summary` should be written to `/tmp/outputs/summary`

### Requirement: Fan-in authoring uses distinct target inputs

The system SHALL help users build fan-in pipelines by making target input ports clear and preventing or warning on duplicate target input bindings.

**Priority**: P0
**Rationale**: Argo cannot bind multiple upstream outputs to the same input parameter, and users need guidance before submission.

#### Scenario: Duplicate target input is visible before save

- **Given** `join.input` is already connected from `left.output`
- **When** the user attempts to connect `right.output` to `join.input`
- **Then** the UI prevents the connection or immediately shows a clear warning
- **And** the invalid shape is not silently accepted as a normal pipeline

#### Scenario: Distinct join inputs are supported

- **Given** a join node has input ports `left` and `right`
- **When** the user connects `left.output` to `join.left` and `right.output` to `join.right`
- **Then** the pipeline is valid and can be saved and run

### Requirement: Standard complex pipeline examples are available

The system SHALL provide reusable complex pipeline examples that users can load as starting points.

**Priority**: P0
**Rationale**: Examples reduce trial-and-error and provide stable regression fixtures for complex workflow behavior.

#### Scenario: Sequential example loads

- **Given** the user opens the pipeline designer
- **When** the user selects the sequential chain example
- **Then** the canvas is populated with a valid multi-step sequential pipeline

#### Scenario: Fan-out/fan-in example loads

- **Given** the user opens the pipeline designer
- **When** the user selects the fan-out/fan-in example
- **Then** the canvas is populated with a valid join pipeline using distinct target inputs

#### Scenario: Observation example loads

- **Given** the user opens the pipeline designer
- **When** the user selects the observation workflow example
- **Then** the canvas is populated with a multi-node workflow suitable for observing logs, duration, and cost
