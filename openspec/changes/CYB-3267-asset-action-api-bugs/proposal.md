# Proposal — CYB-3267

## Why

Two backend bugs surfaced during the DataBrew↔Grace migration. Bug 1 makes
`GET /assets/:id/actions` return 500 for every segment (the Action 时间轴 tab is
unusable). Bug 2 is a stale report — the `duration_sec=0` symptom is already
fixed by CYB-3226 — but the requested change is a sound consistency improvement.

## What Changes

### Modified Capabilities
- **asset-management：Action 列表读取** — `GET /assets/:id/actions` returns the
  asset's actions (currently 500) and each action carries its `task_id`.
- **asset-management：资产创建时长** — `POST /assets` sets `duration_ms` directly
  at the create site (consistency with child-asset creation); user-visible
  `duration_sec` was already correct via CYB-3226 canonicalization.

## Impact
- **Affected code**:
  - `backend/internal/postgres/actions.go` — `scanAction` missing `&a.TaskID` (Bug 1)
  - `backend/internal/usecase/asset/usecase.go` — `Create` sets `DurationMs` (Bug 2, defensive)
- **New APIs**: none (behavior fix to existing endpoints; response shapes unchanged)
- **Dependencies**: none

## Scope
- **In scope**: the two one-line-ish fixes + regression tests + smoke for the actions read path.
- **Out of scope**: any change to action write paths, the `actions` schema, or `duration_ms` canonicalization (CYB-3226 owns it).

## Success Criteria
- [ ] `GET /assets/:id/actions` returns 200 with the correct rows; `task_id` is populated and `tenant_id`/`project_id` are not shifted.
- [ ] A `scanAction` regression test fails if the SELECT column count and Scan destination count diverge again.
- [ ] `POST /assets` returns a correct `duration_sec`/`duration_ms` (regression stays green).
