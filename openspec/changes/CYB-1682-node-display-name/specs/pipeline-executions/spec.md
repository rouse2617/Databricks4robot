# Spec Delta — Pipeline Executions

## Modified Requirement: Workflow DAG node display names

### Scenario: Display component name instead of technical Argo step id
Given a workflow execution was created from a DataBrew pipeline template
And the workflow contains an Argo node or static task named `step-step-1`
And the source pipeline node has component name `cloudrun-e2e-test`
When the user opens the workflow execution detail page
Then the DAG node card primary title SHALL be `cloudrun-e2e-test`
And `step-step-1` SHALL remain available as secondary technical metadata.

### Scenario: Preserve technical node id for diagnostics
Given a DAG node is displayed with component name `cloudrun-e2e-test`
When the user opens logs, Pod diagnostics, resources, or debug actions for that node
Then the frontend SHALL continue sending the original workflow node id/task id expected by the backend.

### Scenario: Fallback when pipeline metadata is unavailable
Given the workflow detail response does not include enough pipeline/template metadata to resolve a component name
When the user opens the workflow execution detail page
Then the UI MAY fall back to the existing Argo display name or node id.
