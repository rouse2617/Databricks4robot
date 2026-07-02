# Pipeline Spec Delta — CYB-1541 Pipeline Run Smoke Fixes

## MODIFIED Requirements

### Requirement: No-asset pipeline runs persist an empty asset list
- **Before**: The system could receive a no-asset run request with an empty asset list but persist the run with `NULL` asset IDs, violating storage constraints.
- **After**: The system SHALL persist no-asset pipeline runs with an explicit empty asset list and SHALL NOT fail due to an asset ID not-null constraint.
- **Reason**: No-asset runs are a valid execution mode and are required for smoke testing reusable pipeline templates before asset selection.

**Priority**: P0 (Critical)
**Rationale**: A persistence failure blocks the primary pipeline run workflow before Argo execution can start.

#### Scenario: No-asset run request creates a durable run
- **Given** a saved pipeline template and an enabled execution target exist
- **When** a client creates a run with `asset_ids` set to an empty list
- **Then** the run is persisted with an empty asset list and does not fail with a database not-null error

#### Scenario: Asset run still preserves selected assets
- **Given** a saved pipeline template and a selected asset exist
- **When** a client creates a run with one or more asset IDs
- **Then** the run is persisted with those asset IDs unchanged

### Requirement: Pipeline form inputs replace stale values
- **Before**: Some pipeline UI inputs could append new text to prior values, causing overlong workflow names and stale asset searches.
- **After**: The pipeline UI SHALL replace input values according to the user's edit operation and reset run-dialog search state when the dialog is reopened.
- **Reason**: Users must be able to correct pipeline names and search different assets without refreshing the page.

**Priority**: P1 (High)
**Rationale**: Stale form state can generate invalid workflow names and makes repeated runs unreliable.

#### Scenario: Pipeline name is replaced before save
- **Given** the design page has a default pipeline name
- **When** the user replaces the name with a new value
- **Then** saving the pipeline uses only the new value

#### Scenario: Run dialog clears stale asset search
- **Given** a user searched for an asset in the run dialog and then closed it
- **When** the user opens the run dialog again
- **Then** the search text, search results, and selected transient assets are cleared

### Requirement: Pipeline runtime objects expose stable IDs in the UI
- **Before**: Pipeline, component, and execution management views could emphasize names while making stable IDs hard to find.
- **After**: The pipeline UI SHALL show asset-style 8-character stable ID labels for saved pipelines, reusable components, and execution records wherever users manage or inspect those objects, while preserving access to the full underlying ID for copy/debug.
- **Reason**: Operators need stable identifiers when debugging duplicate names, API calls, Argo workflows, and persisted run records.

**Priority**: P1 (High)
**Rationale**: Names are user-editable and can collide; IDs are required for reliable debugging and support handoff.

#### Scenario: IDs are visible in management lists
- **Given** saved pipelines, components, and execution records exist
- **When** a user opens their corresponding management views
- **Then** each row or card shows the object's 8-character stable ID label near its name and exposes the full ID for copy/debug

#### Scenario: IDs remain visible in execution detail
- **Given** a user opens an execution detail page
- **When** the detail header and node panels render
- **Then** the stable execution or workflow ID remains visible and copyable where supported
