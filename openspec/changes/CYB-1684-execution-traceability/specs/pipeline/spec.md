# Pipeline Spec Delta

## Modified Requirements

### Requirement: Pipeline Execution Traceability

The pipeline execution ledger SHALL preserve and present enough metadata for users to identify which pipeline definition produced a run.

#### Scenario: Execution list shows pipeline origin

- **Given** a pipeline run exists in the execution ledger
- **When** the user views the execution list
- **Then** each row shows the pipeline name associated with the run
- **And** the row shows the pipeline version associated with the run
- **And** the user does not have to infer execution origin from workflow name alone

#### Scenario: Historical run survives template drift

- **Given** a pipeline run was created from a saved template version
- **And** the current template is later renamed or updated
- **When** the user views the historical run
- **Then** the run still shows the original pipeline traceability metadata captured at execution time

#### Scenario: Execution detail shows origin summary

- **Given** a pipeline run detail page is opened
- **When** the page renders the run summary
- **Then** it shows the pipeline name
- **And** it shows the version or snapshot identity used for that run
- **And** it shows the trigger source for that run

#### Scenario: Batch or asset-triggered runs expose trigger source

- **Given** a run was created from asset-driven flow, batch run flow, manual flow, or API flow
- **When** the user views execution list or detail
- **Then** the UI shows a stable trigger source label for that run

#### Scenario: Failed run exposes failure context in the list

- **Given** a run is in a failed or error terminal state
- **When** the user views the execution list
- **Then** the row provides a visible short failure summary or a direct path to the failure context without requiring blind drilling
