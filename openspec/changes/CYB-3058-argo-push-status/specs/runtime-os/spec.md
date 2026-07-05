# Runtime OS Spec Delta — CYB-3058

## ADDED Requirements

### Requirement: Workflow pushes terminal status to DataBrew

The system SHALL emit a status notification to DataBrew when a pipeline's Argo workflow reaches a terminal phase, so that run status is updated from a push signal rather than solely from polling.

**Priority**: P0 (Critical)
**Rationale**: Polling every 3s with a 100-run scan cap leaves status stale and incomplete at batch scale; pushing terminal status removes the coverage gap and cuts poll load. Terminal push is the highest-value, lowest-cost signal because it is what users wait on and what unblocks downstream batch accounting.

#### Scenario: Workflow success is pushed
- **Given** a run's workflow was submitted with the status-notification hook enabled
- **When** the workflow reaches a `Succeeded` phase
- **Then** DataBrew receives a status notification for that run
- **Then** the run's status becomes the succeeded terminal state and a durable run event is recorded

#### Scenario: Workflow failure is pushed
- **Given** a run's workflow was submitted with the status-notification hook enabled
- **When** the workflow reaches a `Failed` or `Error` phase
- **Then** DataBrew receives a status notification for that run
- **Then** the run's status becomes the failed terminal state and a durable run event is recorded

#### Scenario: Notification delivery fails without blocking the workflow
- **Given** a workflow reaches a terminal phase
- **When** the DataBrew status endpoint is unreachable or returns an error
- **Then** the workflow still completes and is GC'd per its normal lifecycle
- **Then** the run's status is still eventually reconciled by the polling backstop

### Requirement: Idempotent authenticated run status webhook

The system SHALL accept run status notifications on an authenticated endpoint and apply them idempotently, so that duplicate or replayed deliveries do not create duplicate events or corrupt state.

**Priority**: P0 (Critical)
**Rationale**: Push transports deliver at-least-once and can replay; without idempotency the ledger would double-count and status could flap. Authentication is required because the endpoint mutates run state.

#### Scenario: Duplicate delivery is deduplicated
- **Given** a status notification for a run and phase has already been applied
- **When** the identical notification is delivered again
- **Then** no additional durable run event is created
- **Then** the run's status is unchanged by the duplicate

#### Scenario: Unauthenticated call is rejected
- **Given** a status notification without a valid service token
- **When** it is delivered to the webhook endpoint
- **Then** the request is rejected with an unauthorized error
- **Then** no run state is mutated

#### Scenario: Notification for an unknown workflow is rejected safely
- **Given** a status notification whose workflow name maps to no known run
- **When** it is delivered to the webhook endpoint
- **Then** the request is rejected with a not-found error
- **Then** no run state is mutated and no event is written

### Requirement: Observed run status is monotonic

The system SHALL ignore a status observation that would move a run to an earlier phase than its current phase, so that late or out-of-order deliveries cannot regress status.

**Priority**: P0 (Critical)
**Rationale**: Push and poll can race and arrive out of order (e.g. a delayed `Running` after a `Succeeded`); status must only advance so users never see a completed run revert to running.

#### Scenario: Stale non-terminal delivery after terminal is ignored
- **Given** a run has already reached a terminal phase
- **When** a delayed non-terminal (e.g. `Running`) observation for the same run arrives
- **Then** the run's status remains the terminal phase
- **Then** no regressing event is recorded

#### Scenario: Forward progress is applied
- **Given** a run is currently in a running phase
- **When** a terminal observation for that run arrives
- **Then** the run advances to the terminal phase

## MODIFIED Requirements

### Requirement: Run status polling watcher
- **Before**: The system SHALL poll Argo for active run status on a fixed 3-second interval scanning up to a fixed 100 active runs, as the primary means of updating run status.
- **After**: The system SHALL poll Argo for active run status on a configurable interval (defaulting to a lower frequency of 30–60s) and configurable scan limit, serving as a reconcile backstop behind the push path while remaining the authority for correctness (reconciling dropped notifications and backfilling missing ledger events).
- **Reason**: With terminal status arriving via push, high-frequency polling is redundant load; the poller's remaining job is eventual-correctness reconciliation, which tolerates a longer interval.

#### Scenario: Poll interval is configurable
- **Given** a configured watcher interval value
- **When** the backend starts the run status watcher
- **Then** the watcher polls on the configured interval rather than a hardcoded 3s

#### Scenario: Backstop still reconciles a dropped notification
- **Given** a workflow reached a terminal phase but its push notification was never delivered
- **When** the polling backstop next scans that run
- **Then** the run's terminal status and ledger events are reconciled from Argo
