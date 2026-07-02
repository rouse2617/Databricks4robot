# Pipeline Spec Delta — CYB-1536 Asset Validation

## New Requirements

### Requirement: Validate explicit run asset IDs
The system SHALL validate explicit asset ID lists before creating run-like records.

#### Scenario: Pipeline run with missing asset
- **Given** a client submits a pipeline run or deploy request with an unknown asset ID
- **When** the system validates the request
- **Then** it returns `400 INVALID_ARGUMENT` and does not create the run or deployment record.

#### Scenario: Algo run with missing input asset
- **Given** a client creates an algo run with `input_asset_ids`
- **When** any explicit input asset does not exist or is deleted
- **Then** the system returns `400 INVALID_ARGUMENT` and includes the missing asset IDs.

#### Scenario: Run with duplicate explicit assets
- **Given** a client submits explicit asset IDs with the same ID more than once
- **When** the system validates the request
- **Then** it returns `400 INVALID_ARGUMENT` and includes the duplicate asset IDs.

#### Scenario: No-asset flow remains valid
- **Given** a client omits asset IDs or passes an empty list
- **When** the operation supports no-asset or filter-only execution
- **Then** the system accepts the request without requiring explicit assets.

### Requirement: Batch validation
The system SHALL validate explicit asset IDs using a bounded batch lookup instead of one repository call per asset.

#### Scenario: Multiple asset IDs
- **Given** a client sends multiple asset IDs
- **When** the system validates them
- **Then** it performs a batch existence check and reports missing IDs in request order.
