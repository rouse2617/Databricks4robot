## MODIFIED Requirements

### Requirement: Pipeline edge dependency semantics

- **Before**: A canvas edge MAY be translated into an Argo task argument that reads an upstream output parameter, causing deployments to fail when the upstream task does not declare that output.
- **After**: A normal canvas edge SHALL translate to an Argo DAG dependency only, unless an explicit data-binding feature marks the edge as carrying data.
- **Reason**: Users draw edges primarily to define execution order. Data passing must be explicit because many steps do not produce output parameters.

**Priority**: P1 (High)

#### Scenario: Two connected steps run sequentially without output files

- **Given** a pipeline has `step-1 -> step-2`
- **And** `step-1` does not write `/tmp/outputs/output`
- **When** DataBrew deploys the pipeline
- **Then** the Argo DAG task for `step-2` depends on `step-1`
- **And** the task does not contain `{{tasks.step-1.outputs.parameters.output}}`.

#### Scenario: Port metadata remains available

- **Given** a pipeline edge includes source and target handles
- **When** DataBrew builds the workflow
- **Then** the edge still contributes ordering
- **And** port metadata is not converted into task parameters unless explicit data binding is supported.
