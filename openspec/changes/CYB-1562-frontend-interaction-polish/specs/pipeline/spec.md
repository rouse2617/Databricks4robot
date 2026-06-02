# Pipeline Spec Delta

## Modified Requirements

### Requirement: Pipeline Designer Interaction Clarity

The pipeline designer SHALL make component insertion, saving, and deploy preview behavior clear to users.

#### Scenario: Add a component from the component panel

- **Given** the pipeline designer is open
- **When** the user clicks a component in the component panel
- **Then** the component is added to the canvas
- **And** the component control label indicates that click-to-add is supported

#### Scenario: Save a pipeline template

- **Given** the user has a valid pipeline draft
- **When** the user saves it
- **Then** the UI provides a clear success state
- **And** the user can continue editing without losing context unexpectedly

#### Scenario: Preview a deployment

- **Given** the user opens deploy preview
- **When** the dry-run result is available
- **Then** the UI first shows a concise workflow summary
- **And** raw YAML is available as an advanced detail rather than the only view

### Requirement: Workflow Execution Labels

Workflow execution list labels SHALL be displayed with human-readable aliases for known Argo/Event labels.

#### Scenario: Known event labels exist

- **Given** a workflow row has labels such as `events.argoproj.io/sensor`
- **When** the execution list renders
- **Then** the row and filter display a friendly label such as `事件传感器`
- **And** the raw key remains discoverable in a tooltip, copy action, or detail view

### Requirement: Component Detail View

The pipeline component management page SHALL show component details in a readable read-only view.

#### Scenario: User views a component

- **Given** a pipeline component exists
- **When** the user clicks view
- **Then** the modal shows the component's actual values
- **And** it does not present empty disabled form fields as the primary view
