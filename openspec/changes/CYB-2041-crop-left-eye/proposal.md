# Proposal — CYB-2041 Legacy backfill crop repro harness

## Why

Operators need idempotent SQL seeds and a single shell entry point to reproduce legacy backfill result manifests for left-eye, right-eye, and both-eyes crop cases without editing runtime `backend/` or `Frontend/` code.

## What Changes

### Added capabilities
- `backend/scripts/legacy-backfill-{left,right,both}-*.sql` — idempotent `report_manifests` inserts per crop case.
- `scripts/test2-backfill-results-legacy.sh` — dispatches `right_eye | left_eye | both_eyes` with shared env guards.

### Modified capabilities
- None (no application runtime).

## Impact

- **Affected code**: `backend/scripts/`, `scripts/` only
- **New APIs**: None
- **Dependencies**: `psql`, dev credentials (`LEGACY_BACKFILL_DB_URL`, etc.)

## Scope

- **In scope**: Manifest seed SQL + harness; OpenSpec traceability for CYB-2041.
- **Out of scope**: `POST /api/v1/backfill/results` (CYB-2097), workflow execution, UI changes.

## Success Criteria

- [ ] `bash -n scripts/test2-backfill-results-legacy.sh` passes.
- [ ] Each case SQL file is idempotent (`ON CONFLICT` upsert).
- [ ] Harness documents required env vars and fails fast when missing.
