## ADDED Requirements

### Requirement: Legacy backfill manifest repro harness

The repository SHALL provide idempotent SQL seeds and a shell harness so operators can register legacy crop-specific `report_manifests` rows for backfill result reproduction without application code changes.

#### Scenario: Left eye manifest seed is idempotent

- **WHEN** an operator runs `bash scripts/test2-backfill-results-legacy.sh left_eye` with required env vars set
- **THEN** `report.project@1.0.0-backfill-left-eye-only` exists in `report_manifests`
- **AND** re-running the same command does not fail or duplicate rows

#### Scenario: Missing credentials fail fast

- **WHEN** required env vars are unset
- **THEN** the harness exits non-zero with a clear error before connecting to the database
