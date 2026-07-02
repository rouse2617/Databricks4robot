# Runtime OS Spec Delta - CYB-3008

## ADDED Requirements

### Requirement: Run Inspector shows Run metadata

The Run Inspector SHALL expose Run Inputs, Run Outputs, and Runtime reference metadata for DataBrew Runs.

**Priority**: P1 (High)
**Rationale**: Runtime OS defines Config as RunInput and Artifact/Output as RunOutput; users need these projections in the product detail view.

#### Scenario: Run metadata loads

- **Given** a Run detail page is opened by Run id or resolved workflow name
- **When** the Run is resolved
- **Then** the frontend loads Run inputs, outputs, and runtime metadata by Run id
- **Then** the Run Inspector displays the metadata in the page context

#### Scenario: Metadata endpoint failure

- **Given** a Run detail page is opened
- **When** one metadata endpoint fails
- **Then** the DAG, events, asset-node snapshots, logs, and runtime debug remain usable
- **Then** the metadata section shows a local warning or empty state

#### Scenario: External workflow

- **Given** a workflow has no DataBrew Run
- **When** the workflow detail page is opened
- **Then** Run metadata is not shown as product data
- **Then** existing runtime debug views can still be used where available
