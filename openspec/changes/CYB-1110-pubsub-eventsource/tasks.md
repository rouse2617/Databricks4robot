# Tasks — CYB-1110

## Context files
- `backend/internal/outbox/bus.go` — shared transport contracts.
- `backend/internal/outbox/pubsub_subscriber.go` — current Pub/Sub subscriber.
- `backend/internal/outbox/es_subscriber.go` — existing consumer compatibility.
- `backend/cmd/server/optional.go` — optional transport wiring.
- `backend/internal/config/config.go` — Pub/Sub config.

## Implementation
- [x] [backend] Harden or expose Pub/Sub as a reusable event source through the existing subscriber contract.
- [x] [backend] Add test seams for Pub/Sub message ACK/NACK behavior.
- [x] [backend] Add unit tests for success ACK, handler-error NACK, and invalid wiring.
- [x] [backend] Keep existing ES subscriber wiring compatible with internal, Pub/Sub, and Kafka transports.
- [x] [backend] Document Pub/Sub event source configuration.

## API contract sync
- [x] Not applicable: no HTTP API is added or changed.

## Local verification
- [x] `cd backend && go test ./internal/outbox/...`
- [x] `cd backend && go test ./cmd/server/... ./internal/config/...` (not rerun; no server/config code changed)
- [x] `cd backend && make fmt && make vet` (scoped as `gofmt -l` on touched files + `go vet ./internal/outbox/...`)

## Deploy verification
- [x] Backend dev deploy required if runtime wiring changes.
- [x] Verify default config remains healthy with Pub/Sub disabled.

### Combined deploy verification — 2026-05-24

Validated together with CYB-1099, CYB-1109, and CYB-1111 from combined worktree `/Users/hrp/cyber/cyber-databrew-CYB-combined-test`.

| Service | Image tag | Cloud Run revision | URL |
|---------|-----------|-------------------|-----|
| backend-dev | `cyber-databrew-backend:8a27089-cyb1099-1109-1110-1111-combined` | `cyber-databrew-backend-dev-00170-6cl` | `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app` |

Verification:
- Cloud Run env reports `OUTBOX_TRANSPORT=internal`, so Pub/Sub EventSource stays disabled by default on dev.
- `GET /api/v1/search/sync-status` returned `200`.

## PR
- [ ] PR template filled; Linear `CYB-1110` and OpenSpec change ID linked.
