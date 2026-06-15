# Decisions — CYB-2041-left-eye-only-repro

## 2026-06-15 — Scope and naming

- **Change id:** `CYB-2041-crop-left-eye` (folder); Linear title may differ — traceability via `CYB-2041` in commits/PR.
- **Repro only:** No `backend/` or `Frontend/` edits; legacy SQL + harness script + OpenSpec.
- **Case key:** `left_eye` (not `crop_left_eye`) to align with existing `right_eye` / `both_eyes` pattern in `test2-backfill-results-legacy.sh`.
- **Report id:** `report.project@1.0.0-backfill-left-eye-only` — distinct from `right-eye-only` and `both-eyes` manifests.

## 2026-06-15 — Harness contract

- Same env guards as other legacy cases: `DATABREW_BASE_URL`, `DATABREW_TOKEN`, `LEGACY_BACKFILL_DB_URL`, `LEGACY_BACKFILL_ANCHOR_ASSET_ID`.
- SQL path: `backend/scripts/legacy-backfill-left-eye-only.sql`.
- Regression: `scripts/regression-test.sh` case `66` (skipped when anchor unset).

## 2026-06-15 — Verification (agent)

- `bash -n scripts/test2-backfill-results-legacy.sh` — pass.
- Full `left_eye` harness run — **not executed** in agent session (requires user dev DB + credentials). User to run after deploy approval per `deploy-before-commit.md`.

## OpenSpec checkpoint

- Artifacts ready: `proposal.md`, `tasks.md`, `specs/backfill/spec.md`, `decisions.md`.
- **Runtime code:** none — no OpenSpec user gate required for SQL/script-only path per `AI-RULES.md` docs/infra-style scope; deploy still gated by user approval for anything touching dev execution.
