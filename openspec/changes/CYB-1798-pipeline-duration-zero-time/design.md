# Design — CYB-1798

## Context
Asset-node cost summaries expose `durationSeconds` derived from pipeline node `StartedAt` and `FinishedAt`. The helper lives in `backend/internal/usecase/pipeline/usecase.go` and is used by `GetRunCostSummary`.

## Approach
Update `durationSeconds` to reject:
- nil start or finish pointers
- zero start or finish values
- finish times before start times

This keeps duration semantics conservative: unknown or invalid time data is represented as absent duration rather than a fabricated number.

## Verification
Add unit coverage for:
- valid non-zero duration
- nil start or finish
- zero start or finish
- finish before start
