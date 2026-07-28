# pipeline Specification Delta

## MODIFIED Requirements

### Requirement: Batch child run ownership
- **Before**: The system SHALL create child pipeline runs for a batch job, but the child run owner could be empty even when the parent batch job had a creator.
- **After**: The system SHALL preserve the parent batch job creator as the owner of automatically submitted child pipeline runs.
- **Reason**: Owner-less child runs break cost attribution, owner-filtered history, and operational traceability.

**Priority**: P1 (High)
**Rationale**: Batch execution remains functional, but missing ownership degrades billing, debugging, and user-facing run discovery.

#### Scenario: Batch child run keeps creator ownership
- **Given** a running batch job was created by a user and has pending child work
- **When** the submitter creates a child pipeline run for one pending item
- **Then** the child run is submitted with that user as its owner

#### Scenario: Legacy owner-less batch remains valid
- **Given** a running batch job has no recorded creator
- **When** the submitter creates a child pipeline run for one pending item
- **Then** the child run submission remains valid and uses an empty owner
