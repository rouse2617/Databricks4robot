# Pipeline Spec Delta — CYB-1272 Asset Env Passthrough

## MODIFIED Requirements

### Requirement: Asset-based pipeline runs expose selected asset context to node containers
- **Before**: Pipeline runs could receive selected asset IDs, but node containers did not have a complete, tested asset context contract for storage path, segment metadata, revision, and structured asset summaries.
- **After**: The system SHALL inject selected asset context into every pipeline node container through stable environment variables during Argo manifest generation.
- **Reason**: Asset-driven pipeline tasks must know which DataBrew assets and storage paths to process without relying on ad hoc platform lookups inside each container.

**Priority**: P1 (High)
**Rationale**: Asset-based processing is the core DataBrew pipeline path; without deterministic runtime context, containers cannot reliably process selected assets.

#### Scenario: Selected assets are injected into every node
- **Given** a pipeline template with two executable nodes and two selected assets with storage metadata
- **When** the system renders the Argo manifest for a run
- **Then** every executable node container has `ASSET_IDS`, `ASSET_COUNT`, indexed `ASSET_<n>_*` variables, `ASSETS_JSON`, `VIDEO_ID`, and `REQUEST_ID`

#### Scenario: Missing asset is rejected before workflow creation
- **Given** a pipeline run request includes an asset ID that does not exist
- **When** the system creates the run
- **Then** the system rejects the request before rendering/submitting the workflow and reports the missing asset as an asset validation error

#### Scenario: No-asset runs stay compatible
- **Given** a pipeline run request has no selected assets
- **When** the system renders the Argo manifest
- **Then** the manifest still includes pipeline compatibility env such as `PIPELINE_DEPLOYMENT_ID` and `REQUEST_ID`, and does not include indexed `ASSET_<n>_*` variables

### Requirement: Dry-run manifests use the same asset env contract as submitted runs
- **Before**: Dry-run validation could diverge from the submitted workflow if asset env assembly was not covered by manifest tests.
- **After**: The system SHALL assemble asset environment variables before dry-run and submitted manifest rendering through the same code path.
- **Reason**: Operators and automated tests must be able to inspect dry-run manifests to confirm selected asset context before submitting real workflows.

**Priority**: P1 (High)
**Rationale**: Dry-run is the lowest-risk way to validate pipeline runtime context and prevent broken Argo submissions.

#### Scenario: Dry-run exposes selected asset env
- **Given** a user performs a dry-run deployment with one selected asset
- **When** the dry-run response returns the rendered manifest
- **Then** the manifest contains the same selected-asset env vars that would be submitted to Argo for a real run

#### Scenario: Real run preserves selected asset IDs
- **Given** a user submits a pipeline run with selected assets
- **When** the run is saved
- **Then** the run record preserves the selected asset IDs and the workflow manifest receives the corresponding asset env vars
