# Pipeline Spec Delta — CYB-1542

## ADDED Requirements

### Requirement: Pipeline templates are versioned snapshots

Every save of a pipeline template with the same name SHALL create a new immutable version snapshot rather than overwriting a previous snapshot.

#### Scenario: Saving a template creates the first version

- **Given** no pipeline template named `daily-ingest` exists
- **When** the user saves a pipeline named `daily-ingest`
- **Then** the saved template SHALL have `version = 1`

#### Scenario: Saving again creates the next version

- **Given** `daily-ingest` has versions `v1` and `v2`
- **When** the user saves `daily-ingest` again
- **Then** the saved template SHALL have `version = 3`
- **And** versions `v1` and `v2` SHALL remain retrievable

### Requirement: Template list shows latest logical templates

The template list API and UI SHALL show one row per logical template name by default, representing that template's latest version.

#### Scenario: Multiple versions appear as one list row

- **Given** `daily-ingest` has versions `v1`, `v2`, and `v3`
- **When** the user opens the pipeline template list
- **Then** the list SHALL show one `daily-ingest` row
- **And** the row SHALL identify `v3` as the latest version
- **And** the UI SHALL make historical versions discoverable

### Requirement: Users can load historical template versions

The UI SHALL allow users to select a historical version and restore its saved pipeline graph.

#### Scenario: Loading an older version restores its graph

- **Given** `daily-ingest v1` has one node
- **And** `daily-ingest v2` has two nodes
- **When** the user selects `v1` in the version selector
- **Then** the editor canvas SHALL show the one-node graph from `v1`

#### Scenario: Saving while viewing history creates a new version

- **Given** `daily-ingest` latest version is `v3`
- **And** the user is viewing `v1`
- **When** the user edits and saves
- **Then** the system SHALL create `v4`
- **And** `v1` SHALL remain unchanged

### Requirement: Runs bind to a selected template version

Deploying or creating a run from a saved pipeline template SHALL bind the run to a specific template version.

#### Scenario: Deploy defaults to latest version

- **Given** `daily-ingest` latest version is `v3`
- **When** the user runs `daily-ingest` without choosing a version
- **Then** the run SHALL use `v3`
- **And** the run record SHALL expose `templateVersion = 3`

#### Scenario: Deploy uses selected historical version

- **Given** `daily-ingest` has versions `v1`, `v2`, and `v3`
- **When** the user chooses `v2` in the run dialog and starts a run
- **Then** the generated workflow SHALL use the pipeline graph from `v2`
- **And** the run record SHALL expose `templateVersion = 2`

### Requirement: Execution UI shows template version where known

Execution list and detail pages SHALL show the template version used by a run when the backend returns version metadata.

#### Scenario: Run has template version metadata

- **Given** a run was created from `daily-ingest v2`
- **When** the user opens execution detail
- **Then** the UI SHALL show `daily-ingest · v2`

#### Scenario: Legacy run lacks template version metadata

- **Given** an older execution record has no `templateVersion`
- **When** the user opens execution list or detail
- **Then** the UI SHALL omit the version instead of showing a guessed value
