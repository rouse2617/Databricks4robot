# Runtime OS Spec Delta — CYB-3118

## MODIFIED Requirements

### Requirement: Pods carry cost-attribution labels
- **Before**: The system SHALL label step pods with `batch-job-id`, `template-id`, and `owner` for GKE cost allocation.
- **After**: The system SHALL label step pods with `owner`, the `run-id` (the run id), and — when the run processes exactly one asset — that `asset-id`, so real GCP cost (BigQuery billing export) is attributed per run and per video (joinable to video duration by asset id). `batch-job-id` and `template-id` labels are dropped: both are derivable from the run id via the app database, and `run-id`+`asset-id` are the minimal keys that give the full picture.

#### Scenario: Single-video run
- **Given** a run deployed with exactly one asset id
- **When** its pods are created
- **Then** they carry `cyber-databrew/run-id` (the run id) and `cyber-databrew/asset-id` (that asset id)

#### Scenario: No-asset or multi-asset run
- **Given** a run with zero or more than one asset
- **When** its pods are created
- **Then** they carry `cyber-databrew/run-id` but omit `cyber-databrew/asset-id`
