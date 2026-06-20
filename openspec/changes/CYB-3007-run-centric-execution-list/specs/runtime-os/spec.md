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
