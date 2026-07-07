# Runtime OS Spec Delta — CYB-3078

## MODIFIED Requirements

### Requirement: Batch jobs finalize and notify autonomously
- **Before**: The system SHALL detect a batch job's terminal status and send its completion notification during a batch read (`GetJob`/node-summary), so completion is only observed when the batch page is opened.
- **After**: The system SHALL detect a batch job's terminal status and send its completion notification autonomously — driven by child-workflow completion events and, independently, by a periodic reconcile of non-terminal batch jobs — so completion is observed without a user opening the batch page, and exactly once.
- **Reason**: Child workflows complete in the background; relying on a page read means the completion push does not fire until someone clicks in.

#### Scenario: Batch completes with no page open (backstop)
- **Given** a batch job whose child workflows all reach a terminal state in the background
- **And** no user opens the batch page and no completion hook fires
- **When** the periodic reconcile runs
- **Then** the batch job is marked terminal and its completion notification is sent within the reconcile interval

#### Scenario: Hook fast-path finalizes immediately
- **Given** a batch child run whose exit hook pushes a terminal status
- **When** the run-status webhook is handled
- **Then** the parent batch is synced and, if now terminal, its completion notification is sent without waiting for the reconcile

#### Scenario: Notified exactly once
- **Given** both the webhook cascade and the periodic reconcile observe the same batch reaching terminal
- **When** they both attempt to notify
- **Then** the completion notification is sent exactly once
