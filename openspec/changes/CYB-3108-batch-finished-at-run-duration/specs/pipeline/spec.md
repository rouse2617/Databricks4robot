# Pipeline Spec Delta — CYB-3108

## ADDED Requirements

### Requirement: Batch jobs stamp a completion time on every terminal transition

A batch job SHALL record `finished_at` whenever its status transitions to a
terminal state (`completed` or `failed`), regardless of which finalization path
performs the write (synchronous submission failure, async progress reconcile, or
pilot completion).

#### Scenario: Async-completed batch has a completion time

- **Given** a batch whose subtasks all finish and the reconcile derives `completed`
- **When** the job status is persisted
- **Then** `finished_at` SHALL be set (and not overwritten if already set)

### Requirement: Batch list shows real run duration, not wall-clock

The batch task list SHALL present 耗时 as the subtask run span (earliest subtask
start → latest subtask finish), which excludes submit/queue/pause waiting, rather
than `finished_at − created_at`.

#### Scenario: Duration excludes queue time

- **Given** a completed batch created at 09:00 whose subtasks ran 09:55–10:00
- **When** the list renders 耗时
- **Then** it SHALL show ~5m (run span), not ~1h (wall-clock since creation)

#### Scenario: Completion time falls back to subtask finish

- **Given** a terminal batch whose `finished_at` was never stamped
- **When** the list renders 完成时间
- **Then** it SHALL show the latest subtask finish (`runFinishedAt`)

#### Scenario: Running batches show no duration/completion

- **Given** a running batch (some subtasks finished)
- **When** the list renders 耗时 and 完成时间
- **Then** both SHALL show "—" (partial run span is not presented as final)
