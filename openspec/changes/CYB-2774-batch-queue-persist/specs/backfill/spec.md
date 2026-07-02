## ADDED Requirements

### Requirement: DB-backed work queue for batch item processing
The system SHALL process batch items by polling `backfill_items` from PostgreSQL using `SELECT ... FOR UPDATE SKIP LOCKED` instead of an in-memory channel, so that pending items survive service restarts.

**Priority**: P0 (Critical)
**Rationale**: Without persistence, redeploying the service loses unsubmitted batch items, requiring manual DB recovery.

#### Scenario: Worker polls and claims next pending item
- **Given** a batch job with items in `pending` status
- **When** a worker goroutine polls for the next item
- **Then** it SHALL claim exactly one item atomically using `FOR UPDATE SKIP LOCKED`
- **And** the item SHALL be updated to `running` status with `started_at` set to the current time

#### Scenario: Service restart preserves pending items
- **Given** a batch job has items in `pending` or `running` status
- **When** the service is redeployed
- **Then** after restart, the pending items SHALL remain in the database and be picked up by new worker goroutines
- **And** the batch SHALL eventually reach a terminal state

### Requirement: Lease timeout for crash recovery
The system SHALL reclaim items stuck in `running` status beyond a configurable lease timeout, preventing permanent stalls from worker crashes.

**Priority**: P0 (Critical)
**Rationale**: Without lease recovery, a worker crash during submission leaves items permanently stuck in `running` status, blocking batch completion.

#### Scenario: Worker crashes mid-submission
- **Given** a worker claimed an item and set it to `running` with `started_at`
- **When** the worker crashes and the item's `started_at` exceeds the lease timeout
- **Then** a recovery goroutine SHALL reset the item to `pending`
- **And** increment the `attempts` counter

#### Scenario: Paused batch items are not reclaimed
- **Given** a batch job has been paused by the user
- **When** the stale reaper scans for stuck items
- **Then** it SHALL skip items belonging to paused or cancelled batch jobs

### Requirement: Startup recovery for interrupted batches
The system SHALL scan for incomplete batch jobs on service startup and resume processing.

**Priority**: P1 (High)
**Rationale**: After a redeploy, the new process must pick up where the old one left off without manual intervention.

#### Scenario: Incomplete batch resumes after restart
- **Given** a batch job with status `running` and items in `pending` status
- **When** the service starts
- **Then** the service SHALL automatically start worker goroutines for the incomplete batch
- **And** begin processing pending items

## MODIFIED Requirements

### Requirement: Batch item creation with attempts tracking
The system SHALL track the number of processing attempts per batch item to detect and prevent infinite crash loops.

- **Before**: `backfill_items` has no `attempts` column; a crashed item can be retried indefinitely
- **After**: `backfill_items` SHALL have an `attempts` column; items exceeding `max_attempts` SHALL be marked as `failed`
- **Reason**: Prevents infinite retry loops when a worker crashes repeatedly on the same item
