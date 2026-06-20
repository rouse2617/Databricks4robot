# Runtime OS Spec Delta - CYB-3010

## ADDED Requirements

### Requirement: Batch jobs are parent Runs
The system SHALL create a parent Run root for every new batch job.

**Priority**: P0 (Critical)
**Rationale**: Batch is a product execution tree, so it must have a Run identity that can own children, status, diagnostics, and future controls.

#### Scenario: New batch has a parent Run
- **Given** a user creates a batch execution
- **When** the batch is accepted by the system
- **Then** a parent Run exists for the batch
- **Then** clients can open the parent Run children view without needing a separate batch-only identity

#### Scenario: Historical batch without parent Run remains readable
- **Given** a historical batch has child Runs but no parent Run root
- **When** a client requests the batch children view
- **Then** the system still returns child Runs using legacy metadata fallback
- **Then** the request does not fail only because the parent Run root is missing

### Requirement: Run relationships are durable facts
The system SHALL persist Run parent-child relationships as durable Run facts.

**Priority**: P0 (Critical)
**Rationale**: Batch, retry, resubmit, and rerun lineage cannot depend only on transient event payload projection or legacy batch columns.

#### Scenario: Batch child relation is persisted
- **Given** a child Run is created for a batch
- **When** the Run is saved
- **Then** the system persists a durable relationship from the batch parent Run to the child Run

#### Scenario: Rerun relation is persisted
- **Given** a user reruns an existing Run
- **When** the new Run is created
- **Then** the system persists a durable relationship from the source Run to the new Run

#### Scenario: Relation writes are idempotent
- **Given** the same Run materialization is retried
- **When** the system writes the same relationship again
- **Then** the relationship is not duplicated

### Requirement: Run inputs are durable facts
The system SHALL persist product Run inputs when a Run is created.

**Priority**: P0 (Critical)
**Rationale**: Users need immutable audit of assets, configs, runtime target, and parameters even when configs change later or runtime objects expire.

#### Scenario: Asset and config inputs are persisted
- **Given** a Run is created with asset and config dependencies
- **When** the Run is saved
- **Then** the system persists those dependencies as Run inputs
- **Then** raw config content is not exposed through the Run input view

#### Scenario: Runtime target and parameter inputs are persisted
- **Given** a Run is created with a runtime target and parameter values
- **When** the Run is saved
- **Then** the system persists target and parameter facts as Run inputs

#### Scenario: Historical Run inputs remain readable
- **Given** a historical Run has no durable input rows
- **When** a client requests the Run input view
- **Then** the system derives inputs from legacy Run metadata where possible

### Requirement: Batch aggregate health exposes child failures
The system SHALL expose aggregate health when a batch is still active but already has failed or blocked child Runs.

**Priority**: P1 (High)
**Rationale**: A batch with many active children and many failures should not look healthy only because some children are still running.

#### Scenario: Running batch with failures is degraded
- **Given** a batch has running children and failed children
- **When** the client reads the batch Run tree summary
- **Then** the aggregate status can remain running
- **Then** the health indicator reports that failures are already present

#### Scenario: Blocked children affect health
- **Given** a batch has blocked child Runs
- **When** the client reads the batch Run tree summary
- **Then** the health indicator reports blocking even if the aggregate status is pending or running

## MODIFIED Requirements

### Requirement: Run children are resolved from durable facts first
- **Before**: Run children were derived from `batch_job_id` metadata and RunEvent payload projection.
- **After**: The system SHALL resolve Run children from durable relationship facts first and use legacy projection only as fallback.
- **Reason**: Durable relations are required for stable Run lineage and future batch/backfill/rerun controls.

#### Scenario: Durable relations win
- **Given** a parent Run has durable relationship facts
- **When** a client requests the parent children
- **Then** the response is built from durable facts rather than re-derived legacy metadata

### Requirement: Run inputs are resolved from durable facts first
- **Before**: Run inputs were projected from assets, target snapshots, and PipelineJSON.
- **After**: The system SHALL resolve Run inputs from durable input facts first and use legacy projection only as fallback.
- **Reason**: Inputs are part of Run audit and must survive config evolution and projection code changes.

#### Scenario: Durable inputs win
- **Given** a Run has durable input facts
- **When** a client requests the Run inputs
- **Then** the response is built from durable facts rather than re-derived PipelineJSON
