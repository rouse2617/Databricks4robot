# Current work context

Human-updated scratchpad for compact recovery. All agents should read this after context compaction alongside [`docs/agents/AI-RULES.md`](../../docs/agents/AI-RULES.md).

---

## Active issue

- **Linear:** CYB-1098
- **Title:** D-05: feat(api): GET /audit/lineage-search compliance
- **Branch:** `feat/CYB-1098-lineage-search`
- **Worktree:** `/Users/hrp/cyber/cyber-databrew-CYB-1098`

## Scope

- [ ] docs/infra/governance only
- [x] backend
- [ ] Frontend
- [ ] sdk
- [ ] dagster

## OpenSpec

- **Change dir:** `openspec/changes/CYB-1098-lineage-search`
- **OpenSpec OK:** Approved in chat ("可以的") on 2026-05-23

## Off-limits (require explicit approval in chat)

- [ ] `backend/migrations/`
- [ ] `backend/internal/middleware/auth*`
- [ ] `backend/internal/outbox/`
- [ ] `.env*` / credentials

## API contract sync

New public HTTP API contract for `GET /api/v1/audit/lineage-search`; must sync OpenAPI, api-guide, smoke, and spec delta in the same change.

## Deploy status

- **Required:** Yes (backend runtime)
- **Dev deployed:** Yes — backend image `us-central1-docker.pkg.dev/green-valley-442103/rick-cyber-databrew-images/cyber-databrew-backend:ed7886c-cyb1098`
- **Revision:** `cyber-databrew-backend-dev-00166-tck`
- **URL:** `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app`
- **Verified:** Yes — targeted lineage success, empty result, invalid direction, and invalid depth passed on dev. A complex dev fixture (`run=z646d`, source `247HX7ab`) verified default dependency traversal, depth limiting, cycle bounding, relation type filtering, explicit `revision_of`, upstream multi-parent traversal, and `both` direction behavior through the deployed API. `api-guide-smoke.sh` passed the CYB-1098 lineage success check; only root `/healthz` returned Cloud Run 404, matching the known non-business smoke issue.

## Decisions to sync

- Complete and harden the existing `/audit/lineage-search` skeleton instead of adding a duplicate endpoint.
- Trace dependency/structural `asset_relations` relation types by default and return relation metadata.
- SDK and Frontend are out of scope unless the user expands CYB-1098.

---

_Update the sections above when starting a new CYB issue. Clear "Active issue" constraints after merge._
