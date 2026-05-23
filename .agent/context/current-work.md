# Current work context

Human-updated scratchpad for compact recovery. All agents should read this after context compaction alongside [`docs/agents/AI-RULES.md`](../../docs/agents/AI-RULES.md).

---

## Active issue

- **Linear:** CYB-1124
- **Title:** fix(api): harden delivery C2/index atomicity and algo-runs pagination follow-up
- **Branch:** `fix/CYB-1124-delivery-algoruns-followup`
- **Worktree:** `/Users/hrp/cyber/cyber-databrew`

## Scope

- [ ] docs/infra/governance only
- [x] backend
- [ ] Frontend
- [ ] sdk
- [ ] dagster

## OpenSpec

- **Change dir:** `openspec/changes/CYB-1124-delivery-algoruns-followup`
- **OpenSpec OK:** Approved in chat ("开始") on 2026-05-23

## Off-limits (require explicit approval in chat)

- [ ] `backend/migrations/`
- [ ] `backend/internal/middleware/auth*`
- [ ] `backend/internal/outbox/`
- [ ] `.env*` / credentials

## API contract sync

Existing HTTP APIs are behaviorally corrected. Update docs/smoke and OpenAPI only where response/error semantics need clarification.

## Deploy status

- **Required:** Yes (backend runtime)
- **Dev deployed:** Yes — backend image `us-central1-docker.pkg.dev/green-valley-442103/rick-cyber-databrew-images/cyber-databrew-backend:a15de1f-cyb1124`
- **Revision:** `cyber-databrew-backend-dev-00164-vhr`
- **URL:** `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app`
- **Verified:** Yes — targeted algo-runs pagination, delivery duplicate add-items/idempotency, and delivery cancel/index smoke passed on dev. `api-guide-smoke.sh RUN_WRITES=1` passed business checks with only root `/healthz` returning Cloud Run 404.

## Decisions to sync

- CYB-1124 split from completed CYB-1123 to preserve audit trail.
- SDK out of scope unless public SDK-callable shapes change.
- Local backend Tier L passed: targeted gofmt, `go vet ./...`, `go test ./...`.

---

_Update the sections above when starting a new CYB issue. Clear "Active issue" constraints after merge._
