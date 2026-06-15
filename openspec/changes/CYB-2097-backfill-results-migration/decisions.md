# Decisions — CYB-2097

## 2026-06-15 — OpenSpec package from Linear + repo inventory

- **Trigger:** User requested OpenSpec for CYB-2097 (backfill results migration / visibility chasm).
- **Scope M1:** HTTP upload → `algo_run_results` staging → UI visibility; not full Databricks scheduler migration.
- **Overlap:** CYB-258 (rollup/rerun) and CYB-259 (node drilldown) remain separate changes; CYB-2097 references their APIs but does not duplicate spec files.
- **Checkpoint:** Proposal + design + tasks + spec delta ready. **Stop** before `backend/` / `Frontend/` edits until user confirms 「OpenSpec OK，继续」.

## 2026-06-15 — Staging store

- **Decision:** Keep `algo_run_results` as M1 staging (ADR-1 in design.md).
- **Deferred:** Heavy blobs to GCS-only pointers; webhook on bucket finalize.

## 2026-06-15 — Script alignment

- `scripts/upload-deface-backfill-10.sh` and `AlgoProject/backfill_upload_results.py` are **reference implementations**, not SSOT for API shape — OpenAPI + api-guide win.

## 2026-06-15 — Agent verification

- OpenSpec artifacts only this session; no runtime deploy.
- Next implementer: read `context-files.md`, run `scripts/smoke-backfill-results-api-dev.sh` after handler work.

## 2026-06-15 — User checkpoint approved

- **User:** 「OpenSpec OK，继续」（侧栏确认 CYB-2097 实现）
- **Unblocks:** `backend/`, `Frontend/`, `api/openapi.yaml`, smoke scripts
- **Still gated:** `backend/migrations/` for `algo_run_results` — off-limits per `AI-RULES.md`; needs explicit user approval before DDL
- **Repo note:** `algo_run_results` / `report_manifests` not in current `backend/migrations/`; design ADR-1 assumes legacy/dev tables — confirm schema owner before migration PR
