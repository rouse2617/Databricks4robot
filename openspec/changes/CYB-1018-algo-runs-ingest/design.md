# Design — CYB-1018 (tracer bullet)

## Migration 031

1. `CREATE TABLE algo_runs` per unified catalog §5 (generated `duration_ns`, indexes).
2. Null legacy orphan `run_id` on `asset_algo_latest`, `actions`, `asset_eval_results`, `asset_tags`.
3. Add `fk_*_run` constraints `ON DELETE SET NULL`.
4. Enable deferred `fk_atags_run` from CYB-1015.

## API (P1 tracer)

| Method | Path | Effect |
|--------|------|--------|
| POST | `/api/v1/algo-runs` | Insert `pending`; client supplies `run_id` (16 alphanumeric) |
| POST | `/api/v1/algo-runs/{run_id}/start` | `pending` → `running`, `started_at` |
| POST | `/api/v1/algo-runs/{run_id}/finish` | → `ok` or `failed` + stats |
| GET | `/api/v1/algo-runs/{run_id}` | Read row |

Deferred: list, cancel, `affected-assets`.

## Per-asset algo integration

- `StartAlgo` / `FinishAlgo`: if `run_id` present, must exist in `algo_runs`.
- `FinishAlgo` success path: append `algo_run_applied` with `run_id`, `algo_name`, `algo_version`, `new_status`.

## Idempotency

- Create: duplicate `run_id` → 409 unless same payload (row_version check deferred; 409 on PK conflict).
