# Tasks — CYB-2041 Legacy backfill crop repro

## 1. OpenSpec + ops artifacts

- [x] proposal.md
- [x] tasks.md
- [x] specs/backfill/spec.md
- [x] decisions.md

## 2. SQL seeds

- [x] `backend/scripts/legacy-backfill-left-eye-only.sql`
- [x] `backend/scripts/legacy-backfill-right-eye-only.sql`
- [x] `backend/scripts/legacy-backfill-both-eyes.sql`

## 3. Harness

- [x] `scripts/test2-backfill-results-legacy.sh` — `right_eye | left_eye | both_eyes`
- [x] Shared env guards: `DATABREW_BASE_URL`, `DATABREW_TOKEN`, `LEGACY_BACKFILL_DB_URL`, `LEGACY_BACKFILL_ANCHOR_ASSET_ID`

## 4. Verify

- [x] `bash -n scripts/test2-backfill-results-legacy.sh`
- [ ] Full harness run on dev DB (operator; requires credentials)

## Out of scope

- Runtime handler / UI (CYB-2097)
- `scripts/regression-test.sh` case 66 (follow-up if CI gate needed)
