# Runtime OS Spec Delta - CYB-3006

## ADDED Requirements

### Requirement: Product rerun creates a new Run
The system SHALL expose product rerun as a distinct operation from runtime retry and resubmit.

**Priority**: P0 (Critical)
**Rationale**: Runtime OS must avoid overloading "retry" for both runtime retry and full product rerun.

#### Scenario: Rerun succeeds
- **Given** a source Run exists with a reusable Run spec
- **When** the client calls `POST /api/v1/runs/{id}/rerun`
- **Then** the system creates a new Run from the source Run spec
- **Then** the response is `201` with the new Run
- **Then** RunEvent ledger records the requested operation on the source Run and a created event on the new Run

#### Scenario: Rerun fails
- **Given** a source Run exists but new Run creation fails
- **When** the client calls `POST /api/v1/runs/{id}/rerun`
- **Then** the source Run ledger records a rerun failed event with the failure reason
- **Then** the client receives the existing mapped product error

#### Scenario: Retry remains runtime-level
- **Given** a source Run exists with a runtime reference
- **When** the client calls `POST /api/v1/runs/{id}/retry`
- **Then** the system performs runtime retry on the same Run and does not create a new Run
