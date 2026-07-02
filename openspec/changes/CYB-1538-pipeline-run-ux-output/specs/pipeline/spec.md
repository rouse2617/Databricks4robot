# Pipeline Spec Delta — CYB-1538 Pipeline Run UX Output

## New Requirements

### Requirement: Saved pipeline runs use first-class run API
The frontend SHALL create saved pipeline executions through the first-class pipeline-run endpoint.

#### Scenario: No assets selected
- **Given** a user runs a saved pipeline without selecting assets
- **When** the frontend submits the run
- **Then** the request body includes `asset_ids: []`.

### Requirement: No-asset runs persist as empty arrays
The backend SHALL persist no-asset pipeline runs with a non-null empty asset list.

#### Scenario: Empty asset list
- **Given** a run request contains `asset_ids: []`
- **When** the backend saves the pipeline run
- **Then** `pipeline_runs.asset_ids` is stored as an empty text array and `no_asset_run` is true.

### Requirement: Output files are required only when meaningful
The transpiler SHALL declare Argo file output parameters only for consumed outputs or components that explicitly write those output paths.

#### Scenario: Unconsumed default output
- **Given** a node has an output port that is not consumed
- **And** the component does not reference `/tmp/outputs/<port>`
- **When** the workflow is generated
- **Then** the node template does not declare that output parameter.

#### Scenario: Explicit output file
- **Given** a component command writes `/tmp/outputs/output`
- **When** the workflow is generated
- **Then** the node template declares the `output` parameter and prepares the output directory for shell commands.
