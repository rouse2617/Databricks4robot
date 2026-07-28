# CYB-3715b: contract sync — advertise 7 flatten fields in openapi + api-guide + smoke

## Context

CYB-3715 landed the runtime plumbing (migration + backend filter/facet whitelist) in PR #526 + #527. The public API contract does not yet advertise the new fields:

- `api/openapi.yaml#Asset` schema — 7 flatten fields absent
- `docs/review/api-guide.md` — 顶层字段 section stops at `updated_at`; readers still see `mcap.<col>` as the only path
- `scripts/api-guide-smoke.sh` — no regression assertion for the flatten filter path, so a future refactor could silently break it and CI would not catch it

## Change

Doc + smoke only. No runtime code touched.

1. `Asset` schema in openapi gets 7 new nullable string properties with `description: "CYB-3715: mirrored from mcap_files.<col>."` (source_platform notes the metadata JSONB lift).
2. `api-guide.md` 顶层字段 section adds a CYB-3715 row listing the 7 fields + example curl. Note that mcap.* remains for callers that want the subquery path.
3. `api-guide-smoke.sh` gains a CYB-3715 regression pack analogous to the CYB-3713 keyword pack — asserts HTTP 200 and non-zero filtered total for `camera_model` and `source_platform` (both populated on dev).

## Non-goals

- Subscriber ES doc projection (CYB-3715c)
- Frontend facet chips (CYB-3715d)
- New Linear ticket for C (metadata fulltext) — separate work
