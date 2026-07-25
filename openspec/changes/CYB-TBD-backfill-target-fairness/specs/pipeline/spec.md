# pipeline Specification Delta

## MODIFIED Requirements

### Requirement: Batch submitter candidate fairness
- **Before**: The system SHALL select a fixed global oldest-first window of batch jobs before resolving execution targets and clusters.
- **After**: The system SHALL select submittable batch job candidates fairly across execution targets before resolving clusters for dispatch.
- **Reason**: A paused or saturated target can otherwise monopolize the global candidate window and starve healthy targets.

**Priority**: P1 (High)
**Rationale**: This failure does not duplicate execution, but it can indefinitely delay healthy work and makes per-cluster dispatch isolation ineffective under skewed queues.

#### Scenario: Healthy target is not hidden behind older deferred target
- **Given** many older pending jobs for one target are deferred by pause or backpressure
- **When** a younger pending job for another healthy target exists
- **Then** the submitter includes the healthy target in the same cycle and may submit its work

#### Scenario: Missing target remains eligible
- **Given** a running batch job has pending work but no explicit execution target
- **When** the submitter selects candidates
- **Then** the job remains eligible through the default target bucket
