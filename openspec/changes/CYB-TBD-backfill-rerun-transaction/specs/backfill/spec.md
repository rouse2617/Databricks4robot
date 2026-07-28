# backfill Specification Delta

## MODIFIED Requirements

### Requirement: Rerun re-queues only successfully re-submitted items

- **Before**: `Rerun` flipped every matched item to `pending` up front (one
  batch UPDATE that also cleared `error_message`), then looped submit. Items
  whose submit failed stayed `pending`; when every submit failed, the job was
  never moved to `running` (stayed terminal) while all items dangled at
  `pending`. The submitter only scans `running`/`pilot_running` jobs, so those
  items were never dispatched and never re-failed, and their failure reason had
  been wiped.
- **After**: `Rerun` re-queues an item (flip to `pending`, clear
  `error_message`/`started_at`/`finished_at`) **only after** its own re-submit
  succeeds. A submit that fails leaves the item untouched (keeps its `failed`
  status and `error_message`) and records it under `skipped`. When every submit
  fails, no item is moved, so a terminal job never coexists with dangling
  `pending` items.
- **Reason**: The upfront flip created a window where a terminal job carried
  `pending` items the submitter would never pick up — a permanent leak that
  also destroyed the original failure reason.

**Priority**: P1 (High)
**Rationale**: Data-consistency defect; retried items can be silently stranded.

#### Scenario: All re-submits fail leaves no dangling pending

- **Given** a terminal (`failed`) job whose items are all `failed` with a recorded error
- **When** a rerun is requested and every `UpsertBatchSubtaskRun` fails
- **Then** each item keeps `status = failed` and its `error_message`
- **And** the job stays terminal (never flipped to `running`)
- **And** the result reports `retriedCount = 0` with status `failed`

#### Scenario: Partial success re-queues only the winners

- **Given** a job with two `failed` items
- **When** a rerun re-submits one item successfully and the other fails
- **Then** the succeeded item becomes `pending` bound to a fresh run and the job moves to `running`
- **And** the failed item keeps `status = failed` and its `error_message`
- **And** the result reports `retriedCount = 1` with status `partial_success`

### Requirement: Rerun grants a fresh submit-attempt budget

- **Before**: `PrepareItemsForRerun` reset `status`/`error_message`/
  `started_at`/`finished_at` but not `submit_attempts`. An item poisoned by the
  submit-attempt cap (CYB-3678) kept its exhausted counter through a rerun, so
  the first transient submit failure re-poisoned it immediately.
- **After**: `PrepareItemsForRerun` also resets `submit_attempts = 0`, matching
  the DLQ retry path (`ResetFailedItems`). A human rerun earns a full retry
  budget.
- **Reason**: Rerun is an explicit human retry; without a counter reset the
  retry budget is effectively zero for previously poisoned items.

#### Scenario: Rerun clears the durable attempt counter

- **Given** an item whose `submit_attempts` reached the cap and was moved to `failed`
- **When** the item is prepared for rerun
- **Then** the reset SQL sets `submit_attempts = 0` (alongside `status = 'pending'`)
