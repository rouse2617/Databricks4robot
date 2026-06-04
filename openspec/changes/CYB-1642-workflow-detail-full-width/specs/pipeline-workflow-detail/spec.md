# pipeline-workflow-detail Spec Delta

## Modified Requirements

### Requirement: workflow execution detail uses workbench-width layout

The workflow execution detail page SHALL use the available application work area on desktop viewports.

#### Scenario: wide desktop workflow detail

- **Given** a user opens a workflow execution detail page on a wide desktop viewport
- **When** the DAG view is rendered
- **Then** the page content SHALL not be constrained to a centered 1400px container
- **And** the DAG, node detail table, and event timeline SHALL use the available horizontal work area
- **And** the content SHALL retain compact internal padding.

#### Scenario: standard desktop workflow detail

- **Given** a user opens a workflow execution detail page on a standard desktop viewport
- **When** the DAG view is rendered
- **Then** existing header actions, summary cards, DAG controls, node detail table, and event timeline SHALL remain visible and usable.
