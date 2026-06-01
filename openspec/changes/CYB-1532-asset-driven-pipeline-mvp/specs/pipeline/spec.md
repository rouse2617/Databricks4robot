## ADDED Requirements

### Requirement: Asset-driven pipeline run submission
The system SHALL let users submit a saved or newly authored pipeline as a run against an explicit asset selection and execution target.

**Priority**: P0 (Critical)
**Rationale**: DataBrew's pipeline product must process assets as the primary input unit, not hidden ad-hoc workflow parameters.

#### Scenario: Submit a pipeline run with selected assets
- **Given** a saved pipeline template exists and the selected assets exist
- **When** the user selects those assets, selects an execution target, and starts the run
- **Then** the system creates a run record that includes the pipeline, asset set, target, workflow name, and initial status

#### Scenario: Reject a run with missing assets
- **Given** a saved pipeline template exists and at least one selected asset does not exist
- **When** the user starts the run
- **Then** the system rejects the run before submitting to Argo and shows which asset is invalid

#### Scenario: Submit an explicit no-asset run
- **Given** a user is debugging a pipeline that does not need assets
- **When** the user chooses the no-asset run option and selects an execution target
- **Then** the system creates a run without asset bindings and labels it as a no-asset run

### Requirement: Execution target visibility
The system SHALL expose the execution target used by every pipeline run.

**Priority**: P0 (Critical)
**Rationale**: Multiple clusters and namespaces cannot be managed safely if users cannot see or choose where a workflow runs.

#### Scenario: Select the default dev target
- **Given** the default dev target is available
- **When** the user opens the run dialog
- **Then** the default target is selected and shows its cluster and namespace summary

#### Scenario: Unknown or disabled target
- **Given** a target is unknown or disabled
- **When** the user attempts to start a run with that target
- **Then** the system rejects the run and does not submit a workflow

### Requirement: Run execution record
The system SHALL show pipeline runs as execution records with pipeline name, asset count, execution target, workflow name, status, timestamps, and available actions.

**Priority**: P0 (Critical)
**Rationale**: Users need to understand what ran, where it ran, what assets were involved, and what to inspect next.

#### Scenario: Show a successful asset run
- **Given** a run with selected assets has succeeded
- **When** the user opens the execution records page
- **Then** the record shows success, selected asset count, target, workflow name, and links to run detail and asset list

#### Scenario: Show a failed node run
- **Given** a run failed because a node did not produce a required output
- **When** the user opens the run detail
- **Then** the failed node shows the failure message and links to logs

### Requirement: Node-level logs and resources
The system SHALL expose per-node pod, status, log, and resource information from the run detail flow.

**Priority**: P1 (High)
**Rationale**: Operators must debug steps as Kubernetes pods while keeping the user model at pipeline run and node level.

#### Scenario: Inspect a succeeded node
- **Given** a run node has succeeded
- **When** the user opens that node
- **Then** the system shows pod name, host, status, duration, resource duration, and logs

#### Scenario: Logs unavailable
- **Given** logs cannot be fetched for a node
- **When** the user opens logs
- **Then** the system shows an actionable error without hiding node metadata

### Requirement: Component runtime contract guidance
The system SHALL guide users when a pipeline component declares outputs that require runtime files or callbacks.

**Priority**: P1 (High)
**Rationale**: A component can print successful output and still fail the workflow if it does not satisfy Argo output expectations.

#### Scenario: Component declares a file-backed output
- **Given** a component or node declares an output parameter
- **When** the user edits or runs the component
- **Then** the system shows the expected output contract before submission

#### Scenario: Output file missing at runtime
- **Given** a node finishes its command but does not write the expected output file
- **When** Argo reports the missing output file
- **Then** the run detail surfaces the missing contract as the primary node error

## MODIFIED Requirements

### Requirement: Pipeline management separates templates from executions
- **Before**: The system SHALL show saved pipeline templates and run history in the same management surface.
- **After**: The system SHALL provide separate work surfaces for reusable components, saved pipeline templates, run execution records, and run submission targets.
- **Reason**: Template CRUD and run monitoring are different user jobs and become hard to scan when many runs exist.

#### Scenario: Manage a saved pipeline
- **Given** saved pipeline templates and execution records both exist
- **When** the user opens pipeline template management
- **Then** the user sees template CRUD actions without needing to scan execution history

#### Scenario: Monitor executions
- **Given** many execution records exist
- **When** the user opens execution records
- **Then** the user sees run status, asset count, target, and inspection actions without template edit controls dominating the list
