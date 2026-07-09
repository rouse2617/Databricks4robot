# CYB-3226 Fix segment duration_ms=0 and harden asset write-time validation

## Problem

### P1 — segments persist `duration_ms=0`
Every `asset_type=segment` asset shows 时长 **0.0s** in the assets list, even
though the segments have real time ranges. Sampling 5 segments on dev: all have
`duration_ms=0` while their `start/end_timestamp_ns` are populated with spans of
23s–333s.

Root cause is an ordering bug in `prepAssetForWrite` (`backend/internal/postgres/repos.go`):

- Segments are created via the top-level `usecase.Create`, which sets only
  `DurationSec` (seconds), not `DurationMs` (`usecase.go:862`).
- `prepAssetForWrite` first calls `a.SyncLegacyFields()` (`repos.go:157`), which
  **unconditionally** recomputes `DurationSec = DurationMs/1000` — with
  `DurationMs=0` this clobbers the just-computed seconds value back to 0.
- The subsequent seconds↔ms mirror block (`repos.go:164-169`) then sees
  `DurationSec == 0` and cannot backfill `DurationMs`.
- Net result: `duration_ms` stays at the column default `0`.

Child assets (clip/action/frame/task) go through `CreateChildAsset`, which sets
`DurationMs` directly (`usecase.go:1015`); that survives `SyncLegacyFields`, so
their duration is correct — only segments are affected.

### P2 — write-time validation is too loose
Asset time-range validation is weak: `end > start` is only enforced inside some
usecase paths, and there is no floor on duration, so degenerate / sub-millisecond
segments can be written (and then render as 0.0s).

## Scope

### Fix
1. Canonicalize `duration_ms` in `prepAssetForWrite` **before** `SyncLegacyFields()`:
   when `DurationMs == 0`, derive it from `DurationSec` if set, otherwise from the
   `end_timestamp_ns - start_timestamp_ns` span (mirroring how `SegmentLocator` is
   already derived from the same two fields).

### Write gates (hardening)
2. **Enforce `end_timestamp_ns > start_timestamp_ns`** consistently across all
   asset write entry points (Create, CreateChild, ranges), returning a clear
   validation error (HTTP 400/422) rather than relying on scattered checks.
3. **Reject sub-1ms duration** — if `(end - start) < 1ms` (i.e. `duration_ms`
   would round to 0), reject the write. This directly prevents the 0.0s class of
   bad data. Decision: threshold = 1ms (see design.md).
4. **Parent time-range consistency — warn-only** — when the parent raw_mcap's
   duration is known, log a warning if the segment's *relative* offset
   `[start,end]` falls outside `[0, mcap_file_duration]` (with tolerance). **No
   rejection** in this change — see design.md for the reference-frame finding
   that makes a hard check unsafe today.

## Out of Scope

- **No DB CHECK constraint** — user chose not to add a schema-level range check;
  no migration. (`duration_ms bigint NOT NULL DEFAULT 0` already exists.)
- **No hard parent-range enforcement** — deferred; warn-only now. A follow-up
  issue will design the relative-offset check once all segment producers are
  confirmed to use file-relative offsets and a tolerance is defined.
- **No backfill of the existing ~10,065 segments** — user decision「存量不管」.
  Only new/updated writes get the correct value.
