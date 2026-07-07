## ADDED Requirements

### Requirement: Run 状态刷新不被过时快照倒退覆盖
The system SHALL confirm a run's current persisted status before reviving it as an active/Pending status from a "still waiting for workflow creation" heuristic; if the run has already reached a terminal status (e.g. rejected by the resource guard) since the in-memory snapshot was taken, the system SHALL NOT overwrite that terminal status and message.

**Priority**: P1 (High)
**Rationale**: A run correctly rejected by the resource guard (clear Failed status + message) was being silently reverted to Pending with an empty message by a stale-snapshot "wait for workflow creation" grace period check, making a clean, fast rejection look like an indefinite stuck state.

#### Scenario: A resource-guard-rejected run is not revived by a stale list snapshot
- **Given** a run has just been marked `Failed` by the resource guard with a clear rejection message
- **When** a concurrent or slightly earlier list/refresh path calls `refreshRunStatus` with an in-memory snapshot of that run still showing `Pending`, and Argo reports the workflow not found
- **Then** the system re-checks the run's current persisted status, finds it already `Failed`, and does not overwrite it back to `Pending`

#### Scenario: A genuinely pending run still gets the workflow-creation grace period
- **Given** a run's current persisted status is still `Pending` and its workflow has not yet appeared in Argo
- **When** `refreshRunStatus` observes a NotFound response within the creation grace period
- **Then** the system behaves as before, waiting for the workflow to appear rather than marking it not-found
