# Current work context

Human-updated scratchpad for compact recovery. All agents should read this after context compaction alongside [`docs/agents/AI-RULES.md`](../../docs/agents/AI-RULES.md).

---

## Active issue

- **Linear:** CYB-1100
- **Title:** D-07: feat(api): GET /logical-assets/{id}/ratings-history
- **Branch:** `feat/CYB-1100-ratings-history`
- **Worktree:** `/Users/hrp/cyber/cyber-databrew-CYB-1100`

## Scope

- [ ] docs/infra/governance only
- [x] backend
- [ ] Frontend
- [ ] sdk
- [ ] dagster

## OpenSpec

- **Change dir:** `openspec/changes/CYB-1100-ratings-history`
- **OpenSpec OK:** Approved in chat ("可以的") on 2026-05-23

## Off-limits (require explicit approval in chat)

- [ ] `backend/migrations/`
- [ ] `backend/internal/middleware/auth*`
- [ ] `backend/internal/outbox/`
- [ ] `.env*` / credentials

## API contract sync

New public HTTP API contract for `GET /api/v1/logical-assets/{id}/ratings-history`; must sync OpenAPI, api-guide, smoke, and spec delta in the same change.

## Deploy status

- **Required:** Yes before commit.
- **Dev deployed:** Yes — backend image `us-central1-docker.pkg.dev/green-valley-442103/rick-cyber-databrew-images/cyber-databrew-backend:519257b-cyb1100`
- **Revision:** `cyber-databrew-backend-dev-00168-fxp`
- **URL:** `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app`
- **Verified:** Yes — targeted ratings-history success, no-ratings, non-rating metric exclusion, invalid id 400, and missing logical id 404 passed on dev. `api-guide-smoke.sh` passed CYB-1100 endpoint checks; only root `/healthz` returned Cloud Run 404.

## Decisions to sync

- Complete and harden the existing `/logical-assets/{id}/ratings-history` route/handler skeleton instead of adding a duplicate endpoint.
- CYB-1100 ratings source is `asset_metrics` `rating.*` rows; `asset_algo_latest` remains algorithm state unless scope changes.
- SDK and Frontend are out of scope unless the user expands CYB-1100.

---

_Update the sections above when starting a new CYB issue. Clear "Active issue" constraints after merge._
