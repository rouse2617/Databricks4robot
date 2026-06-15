## ADDED Requirements

### Requirement: Authenticated backfill result upload

The system SHALL expose `POST /api/v1/backfill/results` for registering algorithm output JSON against an asset and report identity, using the same authentication model as other backfill APIs.

#### Scenario: Successful upload persists staging row

- **WHEN** a client sends a valid manifest and result payload with matching `asset_id`, `report_id`, and `version`
- **AND** the asset exists and the caller is authorized
- **THEN** the API returns success
- **AND** a row exists in `algo_run_results` (or documented successor table) keyed by asset and derivative algo key
- **AND** subsequent derivative checks for that asset and report version can observe the uploaded result

#### Scenario: Asset mismatch rejected

- **WHEN** the manifest `asset_id` does not match the request body or resolved asset
- **THEN** the API returns a client error
- **AND** no staging row is created

#### Scenario: Oversized payload rejected

- **WHEN** the result payload exceeds configured size limits
- **THEN** the API returns a client error with a clear message
- **AND** no partial row is left in an inconsistent state

### Requirement: Batch item visibility for result registration

Batch and backfill detail surfaces SHALL reflect whether an item is awaiting upload, has a registered result, or failed validation, without requiring operators to inspect raw workflow logs only.

#### Scenario: Item awaiting result after successful workflow without derivative key

- **WHEN** a batch item’s workflow has finished but the asset lacks the expected derivative or pipeline report key
- **THEN** the item is not silently marked completed for rollup purposes
- **AND** the detail UI can show an actionable state (e.g. awaiting result or failed with reason)

#### Scenario: Upload clears awaiting state

- **WHEN** a valid upload is accepted for that item’s asset and report key
- **THEN** the item’s visible status updates to reflect registered result per product policy
- **AND** batch/backfill counters reconcile on the next completion pass

### Requirement: No silent stub completion paths

Code paths that gate backfill or batch completion on result presence MUST NOT remain as unimplemented stubs (`Implement me`) in production wiring after this change ships.

#### Scenario: Service wiring invokes real upload handler

- **WHEN** the application starts with backfill upload routes enabled
- **THEN** the upload handler is registered and reachable
- **AND** smoke tests can hit the live endpoint on dev
