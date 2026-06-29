## MODIFIED Requirements

### Requirement: Batch worker per-item timeout
The system SHALL apply a per-item execution timeout when processing batch job items, so that a single hanging API call does not block the entire batch indefinitely.

- **Before**: The system SHALL process batch items using a context derived from `context.Background()` with no deadline, allowing individual `CreateRunByTemplateID` / `executeItem` calls to hang forever.
- **After**: The system SHALL process each batch item with a per-call timeout of 60 seconds. If the deploy call exceeds the timeout, the item SHALL be marked as failed and the worker SHALL continue to the next item.
- **Reason**: Without a per-item timeout, a single unresponsive Argo API call holds a worker goroutine permanently, blocking `wg.Wait()` and preventing the batch job from ever reaching a terminal state.

#### Scenario: Argo API call hangs beyond timeout
- **Given** a batch job with 10 items
- **When** the 3rd item's `DeployByTemplateID` call to Argo hangs for more than 60 seconds
- **Then** that item SHALL be recorded as "failed" with an appropriate timeout error message
- **And** the remaining items SHALL continue processing normally
- **And** the batch job SHALL eventually transition to a terminal state (completed/failed)

#### Scenario: Normal Argo API call completes within timeout
- **Given** a batch job with multiple items
- **When** all `DeployByTemplateID` calls complete within 60 seconds
- **Then** all items SHALL be processed as before (completed or failed based on the deploy result)
- **And** the batch job SHALL transition to the appropriate terminal state

#### Scenario: Timeout during backfill rerun (`runItems`)
- **Given** a batch job has been rerun via the Rerun function
- **When** `runItems` encounters a hanging `executeItem` call that exceeds the timeout
- **Then** that item SHALL be marked as failed
- **And** `runItems` SHALL continue processing subsequent items
- **And** `syncJobProgress` SHALL be called after all items complete
