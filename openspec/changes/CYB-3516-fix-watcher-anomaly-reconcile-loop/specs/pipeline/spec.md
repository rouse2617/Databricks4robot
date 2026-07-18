# Pipeline Spec Delta — CYB-3516 Fix Watcher Anomaly Reconcile Loop

## MODIFIED Requirements

### Requirement: Anomaly reconcile 判定"最近观察过"仅依据生命周期时间戳

The system SHALL determine whether a pipeline run has been observed recently — for the purpose of the watcher anomaly reconcile window — using only lifecycle-significant timestamps (`FinishedAt` → `StartedAt` → `CreatedAt`), and SHALL NOT use `UpdatedAt` as a fallback.

**Rationale**: `PipelineRunRepo.Save` unconditionally sets `UpdatedAt = now` on every upsert. If `UpdatedAt` participates in the "recent" check, each reconcile pass re-observes the same aged run, calls `Save`, bumps `UpdatedAt`, and re-enters the reconcile set forever. Removing `UpdatedAt` from the fallback chain breaks the self-perpetuating loop while preserving detection for genuinely recent runs.

#### Scenario: Aged run with only UpdatedAt refresh is no longer treated as recent
- **Given** a pipeline run whose `FinishedAt` and `StartedAt` are more than the observation window ago, and whose `CreatedAt` is also outside the window
- **And** `UpdatedAt` was bumped to now by a prior anomaly reconcile save
- **When** the watcher evaluates `runObservedRecently` for that run
- **Then** the run SHALL be classified as not recently observed and SHALL NOT re-enter the anomaly reconcile set

#### Scenario: Fresh run with recent StartedAt is still recent
- **Given** a pipeline run with `FinishedAt = nil`, `StartedAt` within the observation window
- **When** the watcher evaluates `runObservedRecently`
- **Then** the run SHALL be classified as recently observed

#### Scenario: Run with only CreatedAt inside window is still recent
- **Given** a pipeline run with `FinishedAt = nil`, `StartedAt = nil`, `CreatedAt` inside the observation window
- **When** the watcher evaluates `runObservedRecently`
- **Then** the run SHALL be classified as recently observed
