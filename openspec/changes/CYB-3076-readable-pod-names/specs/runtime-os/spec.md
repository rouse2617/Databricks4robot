# Runtime OS Spec Delta — CYB-3076

## MODIFIED Requirements

### Requirement: Pipeline step pod names are human-readable
- **Before**: The system SHALL name each pipeline step's Argo template `step-<nodeID>`, so the pod name embeds the opaque node UUID.
- **After**: The system SHALL name each pipeline step's Argo template `step-<component-slug>`, appending a short node id (`-<uuid8>`) only when more than one node shares a component slug, so the pod name identifies the business step by its component name while remaining unique.
- **Reason**: Raw pod names (kubectl/GKE/logs) must let an operator tell which step a pod is without cross-referencing the pipeline definition.

#### Scenario: Pod name shows the step
- **Given** a pipeline node whose component is `head-track-pycuvslam`
- **When** the workflow is transpiled and its pod is created
- **Then** the pod name contains `step-head-track-pycuvslam` followed by Argo's hash

#### Scenario: Duplicate component names still transpile
- **Given** two nodes with the same component name
- **When** the pipeline is transpiled
- **Then** each gets a distinct template name (`step-<slug>-<uuid8>`; residual clash falls back to the full node hex) and the pipeline is not rejected

### Requirement: Pod names lead with the run id for run traceability
- **Before**: The system SHALL name each Argo workflow `<pipelineName>-<random-uuid8>`, so a pod prefix does not identify the DataBrew run.
- **After**: The system SHALL name each Argo workflow with the run id (`pipeline_runs.id`), so every pod name leads with the run id and maps directly to `/runs/<id>`.
- **Reason**: Operators looking at a raw pod must reach the corresponding run without a database lookup.

#### Scenario: Pod maps to its run
- **Given** a run with id `677e86d9-…`
- **When** its step pods are created
- **Then** each pod name begins with `677e86d9-…` and the run is reachable at `/runs/677e86d9-…`

### Requirement: Run views work for both new and legacy template-name formats
- **Before**: The system SHALL relate stored run nodes to pipeline-definition nodes by stripping the `step-` prefix to recover the node ID.
- **After**: The system SHALL relate stored run nodes to pipeline-definition nodes by matching against a candidate-key set per node (the readable `step-<slug>-<uuid8>` and the legacy `step-<nodeID>`), so node-summary ordering, workflow runtime info, and step labels are correct for both new runs and historical runs.
- **Reason**: Historical runs stored the legacy format; changing the format must not break their views (no data migration).

#### Scenario: New run summary ordered and labeled correctly
- **Given** a new run with readable template names
- **When** the batch node-summary and run detail are viewed
- **Then** node columns are ordered by the pipeline DAG and steps show their component name

#### Scenario: Legacy run still works
- **Given** a historical run stored with `step-node-<uuid>` node ids
- **When** it is viewed after this change ships
- **Then** its node-summary order, step labels, and runtime info are unchanged (matched via the legacy candidate key)
