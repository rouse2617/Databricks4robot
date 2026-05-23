# Current work context

Human-updated scratchpad for compact recovery. All agents should read this after context compaction alongside [`docs/agents/AI-RULES.md`](../../docs/agents/AI-RULES.md).

---

## Active issue

- **Linear:** CYB-1097
- **Title:** D-04: feat(api): GET /audit/search cross-asset
- **Branch:** `feat/CYB-1097-audit-search`
- **Worktree:** `/Users/hrp/cyber/cyber-databrew`

## Scope

- [ ] docs/infra/governance only
- [x] backend
- [ ] Frontend
- [ ] sdk
- [ ] dagster

## OpenSpec

- **Change dir:** `openspec/changes/CYB-1097-audit-search`
- **OpenSpec OK:** Approved in chat ("ok") on 2026-05-23

## Off-limits (require explicit approval in chat)

- [ ] `backend/migrations/`
- [ ] `backend/internal/middleware/auth*`
- [ ] `backend/internal/outbox/`
- [ ] `.env*` / credentials

## API contract sync

New public HTTP API contract for `GET /api/v1/audit/search`; must sync OpenAPI, api-guide, smoke, and spec delta in the same change.

## Deploy status

- **Required:** Yes (backend runtime)
- **Dev deployed:** Yes — backend image `us-central1-docker.pkg.dev/green-valley-442103/rick-cyber-databrew-images/cyber-databrew-backend:b29c594-cyb1097`
- **Revision:** `cyber-databrew-backend-dev-00165-hk8`
- **URL:** `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app`
- **Verified:** Yes — targeted audit search success and validation errors passed on dev. `api-guide-smoke.sh` passed CYB-1097 checks; only root `/healthz` returned Cloud Run 404.

## Decisions to sync

- Complete and harden the existing `/audit/search` skeleton instead of adding a duplicate endpoint.
- SDK and Frontend are out of scope unless the user expands CYB-1097.

---

_Update the sections above when starting a new CYB issue. Clear "Active issue" constraints after merge._
