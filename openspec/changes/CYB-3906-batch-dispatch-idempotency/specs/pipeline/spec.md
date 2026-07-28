# Pipeline delta — CYB-3906

## MODIFIED Requirements

### Requirement: Batch subtask submission has one stable workflow identity

- **Before**: The system SHALL pre-create a batch subtask run with a business
  placeholder workflow name, while Deploy submits an Argo workflow named with
  the run UUID; automatic recovery may classify the UID-less placeholder as
  stale and create a second attempt.
- **After**: The system SHALL use one stable UUID identity for a batch item's
  initial run, placeholder workflow name, Argo workflow name, automatic retry,
  and AlreadyExists lookup. Automatic recovery MUST reuse that identity, and
  only an explicit operator rerun MAY create a new attempt identity.
- **Reason**: The split identity defeats deterministic-name convergence and
  can execute the same logical item more than once after a crash, delayed UID
  persistence, or overlapping submitter cycles.

**Priority**: P0 (Critical)
**Rationale**: Duplicate pipeline execution can repeat expensive processing and
external side effects while producing conflicting run history.

#### Scenario: First submission reuses the materialized identity
- **Given** a pending batch item with a newly materialized UID-less run
- **When** the submitter performs the first submission
- **Then** it submits the Argo workflow with that same run UUID
- **And** it does not create an abandoned placeholder attempt

#### Scenario: UID-delayed submission converges through AlreadyExists
- **Given** Argo created the UUID-named workflow but the submitter failed before
  persisting or reading back its UID
- **When** a later submitter cycle processes the still-pending item
- **Then** it retries with the same UUID workflow name
- **And** AlreadyExists is treated as success by fetching that UUID-named
  workflow and persisting its UID
- **And** no second workflow is created

#### Scenario: Concurrent submitters converge on one workflow
- **Given** two backend instances select the same pending item concurrently
- **When** neither has yet persisted an Argo UID
- **Then** both resolve the same initial run UUID and workflow name
- **And** Argo contains at most one workflow for that logical attempt

#### Scenario: Legacy UID-less placeholder is normalized
- **Given** a pending historical item linked to a UID-less run whose workflow
  name uses the legacy `-batch-` business format
- **When** the submitter resumes the item
- **Then** the run and item are normalized to the existing run UUID identity
- **And** the system does not create a new visible attempt solely because the
  legacy placeholder has no UID

#### Scenario: Terminal attempts require explicit rerun
- **Given** an item linked to a terminal run
- **When** an automatic submitter cycle observes it
- **Then** the submitter does not mint or deploy a new attempt
- **And** a new identity is created only after an explicit operator rerun
