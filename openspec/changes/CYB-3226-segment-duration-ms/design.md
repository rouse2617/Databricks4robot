# CYB-3226 Design

## Timestamp reference-frame finding (why parent-range is warn-only)

dev sampling (segment vs its parent mcap file):

| segment | seg_start_ns | seg_end_ns | mcap_start_ns | mcap span |
|---|---|---|---|---|
| 019cc1e1 | 1 | 68,533,333,333 (≈68.5s) | 1,768,970,231,215,424,000 (abs epoch) | ≈236.9s |
| 582c3a28 | 466,686,557,000 (≈466s) | 800,052,813,000 (≈800s) | 1,766,630,614,000,000,000 (abs epoch) | ≈2261s |

- **Segment timestamps are file-relative** (nanoseconds from file start, ~0-based).
- **mcap file timestamps are absolute wall-clock epoch** (ns since 1970).

A naive containment check `seg ⊆ [mcap_start, mcap_end]` would reject **100%** of
segments (1 is never ≥ 1.768e18). The only valid comparison is *relative*:
segment offsets within `[0, mcap_file_duration]`.

Even the relative check has edge cases: segment `019cc1e1#2` has
`seg_end ≈ 236.933s` vs parent mcap span `236.9s` — it **overshoots by ~33ms**
(rounding / whole-file slack). So a hard relative check needs a tolerance, and we
must first confirm **every** segment producer uses file-relative offsets.

**Decision:** parent-range consistency ships as **warn-only** logging in this
change (relative comparison, with tolerance, only when `mcap_file_duration > 0`).
Hard enforcement is deferred to a follow-up issue.

## Decisions

- **duration_ms canonicalization location** — in `prepAssetForWrite`, *before*
  `SyncLegacyFields()`. This is the single chokepoint for all asset writes
  (Create/Upsert), so one edit fixes every path. Priority: explicit `DurationMs`
  (children) > `DurationSec` (segments/top-level) > timestamp span (fallback).
- **Duration floor = 1ms** — reject when `(end-start) < 1_000_000 ns`. Rationale:
  this is exactly the span that rounds to `duration_ms=0`, i.e. the bad-data line;
  a larger floor (e.g. 1s) risks rejecting legitimate short segments.
- **Range validation placement** — a shared helper validates `end>start` and the
  1ms floor, called from the usecase write entry points (Create, CreateChildAsset,
  ranges) so the error surfaces as a clean 400/422 at the handler. Repo-direct
  bulk writes (if any bypass usecase) are noted as a follow-up if found during impl.

## API impact

Endpoints affected (validation tightening, new 400 cases):
`POST /assets`, `POST /assets/:id/{clips,actions,frames,tasks}`, and the ranges
create endpoint. No request/response shape change; error envelope unchanged.
Sync: api-guide validation notes + one reject-path smoke + spec delta (Given/When/Then).
