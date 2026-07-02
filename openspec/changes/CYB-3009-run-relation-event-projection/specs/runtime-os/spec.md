# Runtime OS Spec Delta - CYB-3009

## ADDED Requirements

### Requirement: Run children include event-derived relations

The system SHALL expose child Runs created by product operations through `/runs/{id}/children`.

**Priority**: P1 (High)
**Rationale**: Runtime OS represents Batch, retry, resubmit, and rerun as Run Tree relationships. A source Run must show child Runs created from it without requiring a new relation table in this slice.

#### Scenario: Rerun child

- **Given** a source Run is rerun
- **When** the client calls `/api/v1/runs/{sourceRunId}/children`
- **Then** the response includes the rerun child Run
- **Then** the response includes a relation with `relationType = "rerun_of"`

#### Scenario: Resubmit child

- **Given** a source Run is resubmitted
- **When** the client calls `/api/v1/runs/{sourceRunId}/children`
- **Then** the response includes the resubmit child Run
- **Then** the response includes a relation with `relationType = "resubmit_of"`

#### Scenario: Batch children remain available

- **Given** a batch parent Run has batch child Runs
- **When** the client calls `/api/v1/runs/{parentRunId}/children`
- **Then** existing `batch_child` relations remain present
