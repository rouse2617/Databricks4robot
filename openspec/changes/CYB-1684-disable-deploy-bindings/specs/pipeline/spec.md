# Pipeline Spec Delta

## Modified Requirements

### Requirement: Deploy-mode pipeline edges and bindings

Pipeline deployment SHALL treat canvas edges as execution-order dependencies and SHALL NOT require node output parameter artifacts when output artifacts are skipped.

#### Scenario: Old explicit binding is present during deploy

- **Given** a pipeline node has an argument with `from: "upstream.output"`
- **And** the transpiler is called with `SkipOutputArtifacts: true`
- **When** the workflow is generated
- **Then** the downstream DAG task has no task argument referencing `tasks.upstream.outputs.parameters.output`
- **And** the downstream container has no undeclared `inputs.parameters` reference for that argument
- **And** canvas edge dependencies are still emitted as DAG dependencies

#### Scenario: Non-deploy explicit binding remains supported

- **Given** a pipeline node has an argument with `from: "upstream.output"`
- **And** the transpiler is called without `SkipOutputArtifacts`
- **When** the workflow is generated
- **Then** the downstream DAG task receives an argument from the upstream output parameter
- **And** the upstream template declares the matching output parameter
