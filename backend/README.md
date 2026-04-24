# Backend

Go backend service for the data platform, with pluggable storage backend (`bigtable` or `postgres`) and Pub/Sub integration points.

## What

- Single backend process exposing:
  - asset CRUD and segment commit APIs
  - MCAP metadata/finalize APIs
  - delivery commit and history APIs

## How to Run

```bash
make deps
make run
```

## Config

Copy and edit env:

```bash
cp .env.example .env
```

Main variables:

- `STORAGE_BACKEND` (`bigtable` or `postgres`)
- `BIGTABLE_PROJECT`
- `BIGTABLE_INSTANCE`
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` (when postgres mode)
- `PUBSUB_PROJECT`
- `GRACE_TOKEN`

## Bigtable Schema (source of truth)

- Canonical schema doc: `schemas/sql.md` in the `data4cyber` repository
- Backend Bigtable constants/repositories are aligned to that schema:
  - tables: `assets`, `mcap_files`, `deliveries`
  - index tables: `idx_segments_by_file`, `idx_asset_deliveries`, `idx_customer_deliveries`
  - support table: `idempotency_keys`

Bootstrap tables/CFs with:

```bash
make bt-bootstrap
```

## API / Interfaces

- REST routes are wired in `routes/routes.go`
- Handlers are in `internal/handlers/{asset,mcap,delivery}`
- Business orchestration is in `internal/usecase/{asset,...}`
- Bigtable repositories are in `internal/bigtable`
- OpenAPI contract is in `../api/openapi.yaml`

### API Conventions (Phase 0)

- **Auth**: external APIs require `X-Grace-Token` (temporary). `Authorization: Bearer <token>` is also accepted.
- **Request ID**: every response includes `X-Request-ID`. Client can pass one; otherwise server generates it.
- **Errors**: standardized envelope:
  - `code`
  - `message`
  - `request_id`
  - `details` (optional)
- **Idempotency**:
  - `POST /api/v1/deliveries` requires `Idempotency-Key`
  - Reusing same key with different payload returns `409`
  - Bigtable mode requires table `idempotency_keys` with CF `meta`
  - Postgres mode requires table `idempotency_keys` (see `../schemas/pg-phase0.sql`)
- **Pagination**: list endpoints use fixed `page` + `page_size` and return `next_token` when available.
- **Status codes**:
  - `400` invalid request format/arguments
  - `404` resource not found
  - `409` idempotency conflict
  - `422` semantic domain/state errors
  - `500` internal server error

## Directory Structure

- `cmd/`: service entrypoints
- `internal/config/`: env config
- `internal/usecase/`: business/usecase layer (`handler -> usecase -> repo`)
- `internal/bigtable/`: `client.go` + `repos.go` (simplified layout)
- `internal/postgres/`: `client.go` + `repos.go` (simplified layout)
- `internal/models/`: domain models
- `routes/`: route registration

## Development Workflow

```bash
make test
make fmt
make vet
make build
```

Bigtable package coverage (target: 100%):

```bash
go test ./internal/bigtable -coverprofile=/tmp/bt.cov
go tool cover -func=/tmp/bt.cov
```

Postgres package coverage (target: 100%):

```bash
go test ./internal/postgres -coverprofile=/tmp/pg.cov
go tool cover -func=/tmp/pg.cov
```

## Known Limitations

- `mcap/:id/messages` is placeholder (no message decoding yet)
- Some list/filter APIs are currently minimal
- End-to-end tests are not complete yet

## Next Milestones

- Add emulator table bootstrap script
- Add unit tests for repos and handlers
- Add smoke test script for mcap finalize → segment → delivery flow

