# Runtime OS Spec Delta - CYB-3007

## ADDED Requirements

### Requirement: Execution Hub uses Runs as the only product list source

The Execution Hub product list SHALL use DataBrew Run records as its only source for execution rows.

**Priority**: P0 (Critical)
**Rationale**: Runtime OS defines Run as the product execution entity. Argo workflows are runtime debug objects and must not create product execution rows.

#### Scenario: List executions

- **Given** the user opens Execution Hub
- **When** the page refreshes execution rows
- **Then** the frontend calls the Run list API
- **Then** the table rows are projected from returned Runs
- **Then** the frontend does not query runtime workflow listing to build the product list

#### Scenario: Label-filtered list

- **Given** the URL contains a label filter
- **When** the page refreshes execution rows
- **Then** the frontend still uses the Run list API as the product list source
- **Then** live-only runtime workflows do not appear as execution rows

#### Scenario: Runtime debug remains available

- **Given** a Run row has a runtime workflow reference
- **When** the user opens the Run detail page
- **Then** the frontend may use the workflow reference for runtime debug data such as logs, Pod diagnostics, terminal, and resource metrics
- **Then** the workflow reference is not treated as the product list primary key

#### Scenario: Stable Run route

- **Given** the user opens `/runs`
- **When** the frontend renders the Pipeline area
- **Then** the execution records tab is selected by default
- **Then** product navigation to execution lists uses `/runs`
- **Then** product navigation to a known Run detail uses `/runs/{runId}`

### Requirement: Execution Hub metrics are sortable

The Execution Hub product list SHALL expose working sort controls for key execution metrics shown in the table.

**Priority**: P1 (High)
**Rationale**: Operators need to quickly find the slowest, newest, oldest, or costliest Runs while investigating execution behavior.

#### Scenario: Sort execution metrics

- **Given** the execution list contains multiple Run rows
- **When** the user clicks the duration, total cost, created time, or finished time column header
- **Then** the table rows reorder by the selected metric
- **Then** clicking the same sortable header again reverses that metric order

#### Scenario: Sort rows with missing metric values

- **Given** some Runs have missing cost or finished time values
- **When** the user sorts by that metric
- **Then** rows with available metric values remain sortable without crashing the table
- **Then** rows with missing metric values still render their existing placeholder text

### Requirement: Execution Hub exposes summary cost and template search

The Execution Hub product list SHALL expose available Run cost snapshots and SHALL let users find Runs by linked pipeline template name.

**Priority**: P1 (High)
**Rationale**: Operators need the list to support cost triage and to locate historical executions for a known reusable pipeline template.

#### Scenario: Show available summary cost

- **Given** a Run has persisted node cost snapshots
- **When** the execution list loads that Run through the summary list API
- **Then** the response includes the Run total estimated cost
- **Then** the UI shows that cost in the total cost column

#### Scenario: Missing cost snapshot remains explicit

- **Given** a historical Run has no persisted node cost snapshot
- **When** the execution list loads that Run through the summary list API
- **Then** the response leaves the estimated cost absent
- **Then** the UI keeps rendering the placeholder instead of an incorrect zero

#### Scenario: Search by template name

- **Given** multiple Runs were created from saved pipeline templates
- **When** the user searches the execution list using a pipeline template name
- **Then** the Run list API matches Runs linked to that template name
- **Then** the table shows matching Runs even when they are not on the first unfiltered page
