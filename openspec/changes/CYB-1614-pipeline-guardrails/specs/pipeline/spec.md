# Pipeline Spec Delta — CYB-1614 Pipeline Guardrails

## ADDED Requirements

### Requirement: Duplicate fan-in target ports are rejected before Argo submission
The system SHALL reject pipeline definitions where more than one edge binds to the same target node input parameter before submitting a workflow to Argo.

**Priority**: P0 (Critical)
**Rationale**: Argo rejects duplicate input parameter names, so allowing this shape creates failed runs that users cannot diagnose from the designer.

#### Scenario: Duplicate target input is blocked
- **Given** a pipeline contains two edges targeting `join.input`
- **When** the user saves or runs the pipeline
- **Then** the system reports that `join.input` has multiple upstream bindings and does not submit the workflow to Argo

#### Scenario: Distinct target inputs are allowed
- **Given** a pipeline contains edges targeting `join.left` and `join.right`
- **When** the user saves or runs the pipeline
- **Then** the system accepts the pipeline and Argo receives distinct input parameters

### Requirement: Shell script components use a normalized command shape
The system SHALL normalize shell script components so `sh -c` commands store exactly one script body argument when the command already includes `-c`.

**Priority**: P0 (Critical)
**Rationale**: Storing `sh`, `-c`, and the script as arguments for `command=["sh","-c"]` produces invalid container invocation and failed workflows.

#### Scenario: Command already contains `-c`
- **Given** a user edits a component with command `sh -c`
- **When** the component is saved
- **Then** the persisted component args contain the script body only

#### Scenario: Existing valid shell form remains valid
- **Given** a component uses command `sh` and args `-c`, `<script>`
- **When** the workflow is generated
- **Then** the container runs the script unchanged

### Requirement: Output port file contract is explicit and safe
The system SHALL make declared output ports safe by only requiring Argo output files when the output is consumed or explicitly written, and SHALL guide users to write `/tmp/outputs/<port>` when an output must be produced.

**Priority**: P0 (Critical)
**Rationale**: Declared output ports currently imply required files; missing `/tmp/outputs/output` fails the pod even when users only intended to print logs.

#### Scenario: Consumed output requires file output
- **Given** a downstream edge consumes `step.output`
- **When** the workflow is generated
- **Then** the upstream Argo template declares the `output` parameter and prepares `/tmp/outputs`

#### Scenario: Unconsumed default output does not fail a pod
- **Given** a node has an unconsumed default output port and the script does not write `/tmp/outputs/output`
- **When** the workflow runs
- **Then** the pod can succeed without Argo failing on a missing output file

## MODIFIED Requirements

### Requirement: Pipeline run cost summary communicates syncing state
- **Before**: The UI could show empty or partial node cost summary immediately after workflow completion without explaining that watcher/cost data may still be syncing.
- **After**: The UI SHALL distinguish unavailable cost data from recently syncing or partial cost data.
- **Reason**: Cost estimates are derived from observed node/resource duration and may arrive after Argo marks the workflow complete.

#### Scenario: Recently completed run has partial cost data
- **Given** a run has completed and only some node costs have synced
- **When** the user opens the execution list or detail page
- **Then** the UI shows the available cost values and a syncing/partial-data indication

### Requirement: Workflow logs render as readable lines
- **Before**: Argo live log responses could appear visually compressed when prefixes and log lines arrived without normalized line breaks.
- **After**: The UI SHALL render workflow logs as readable separate lines while preserving bounded log metadata.
- **Reason**: Users need to inspect pod work and debug scripts without manually parsing concatenated log text.

#### Scenario: Argo-prefixed logs are displayed readably
- **Given** the logs API returns Argo-prefixed pod log text
- **When** the user opens a node log view
- **Then** each logical log entry is displayed on its own readable line
