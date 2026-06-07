# Proposal — CYB-1798

## Summary
Treat zero pipeline node timestamps as missing data when computing duration seconds.

## Problem
Backend review finding #5 noted that `durationSeconds(startedAt, finishedAt)` only rejects nil pointers and reversed ranges. A non-nil pointer to `time.Time{}` can be produced by defensive model construction or incomplete timestamp propagation. The current helper can then return a bogus multi-decade duration.

## Goals
- Return `nil` duration when either timestamp pointer is nil or points at a zero time.
- Preserve valid duration calculations for non-zero timestamps.
- Add focused backend regression coverage.

## Non-Goals
- Changing persisted timestamp schema.
- Changing active run watcher lifecycle behavior; that is review finding #6.
