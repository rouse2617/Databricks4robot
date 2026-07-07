## ADDED Requirements

### Requirement: 确定性失败的 run 不被复活逻辑倒退覆盖
The system SHALL treat a run that has reached a definitive terminal failure — a `Failed`/`Error` status carrying a real, non-transient error message (e.g. rejected by the resource guard before any workflow is created) — as final, and SHALL NOT regress it back to an active status (`Pending`/`Running`) via any "waiting for workflow creation" or misclassification-recovery heuristic. A terminal state carrying a known stale/transient "workflow unavailable" message is a misclassification, not a definitive failure, and MAY still be recovered.

**Priority**: P1 (High)
**Rationale**: A batch sub-task correctly rejected by the resource guard (Failed + a clear message, no workflow, placeholder `-batch-` name) matched the "pending workflow creation" heuristic and was silently reverted to Pending with the message wiped, leaving it to poll a workflow that can never exist — a clean rejection presenting as an indefinite stuck run.

#### Scenario: A resource-guard-rejected run stays failed across refreshes
- **Given** a batch sub-task run was marked `Failed` by the resource guard with a substantive rejection message and never got an Argo workflow
- **When** the run detail/list path refreshes it (Argo reports the workflow not found, within the creation grace period)
- **Then** the run remains `Failed` with its rejection message, and is not reverted to `Pending`

#### Scenario: A misclassified (TTL-cleanup) terminal run is still recoverable
- **Given** a run is in a terminal status carrying a known stale/transient "workflow unavailable" message (a TTL-cleanup false positive)
- **When** the reconciliation path re-observes it within the stale-age limit
- **Then** the system still revives it, unaffected by the definitive-failure guard
