# CLAUDE.md

Practical project guidance for coding agents working in `data-platform`.

## What This Repo Is

`data-platform` is a single-backend-process asset platform for MCAP-oriented metadata, delivery tracking, and SDK/frontend integration.

- Backend: Go + Gin (`backend/`)
- SDK: Python + httpx + pydantic (`sdk/`)
- Frontend: React + TypeScript (`Frontend/`)
- Orchestration skeleton: Dagster (`dagster/`)

## Current Architecture (Source of Truth)

### Backend runtime

- **Single process only**: `backend/cmd/server/main.go`
- **Router entry**: `backend/routes/routes.go`
- **Storage backend switch**: `STORAGE_BACKEND=bigtable|postgres`

### Backend layering

- `handler -> usecase -> repository`
- Handlers: `backend/internal/handlers/{asset,mcap,delivery}`
- Usecases: `backend/internal/usecase/`
- Repositories:
  - Bigtable: `backend/internal/bigtable/client.go`, `backend/internal/bigtable/repos.go`
  - Postgres: `backend/internal/postgres/client.go`, `backend/internal/postgres/repos.go`

### Data schema sources

- Bigtable schema source: `schemas/sql.md` in the `data4cyber` repository
- OpenAPI source: `api/openapi.yaml`
- PG schema: `schemas/pg-phase0.sql`

## Removed / Disabled Capabilities

These are intentionally removed and should not be reintroduced unless explicitly requested:

- `GCSRawBucket` capability
- `/api/v1/mcap/upload/init`
- `/api/v1/mcap/:id/download-url`
- Multi-process backend split (`asset-service`, `mcap-gateway`, `delivery-service`)

## API Conventions (Must Keep)

- Auth: `X-Grace-Token` (temporary phase-0 auth)
- Request tracing: `X-Request-ID` middleware
- Error envelope: `code`, `message`, `request_id`, `details`
- Idempotency:
  - `POST /api/v1/deliveries` requires `Idempotency-Key`
- Pagination:
  - `page`, `page_size`, `next_token` style must stay consistent

## Development Rules

1. **Keep interfaces stable** between handlers/usecases/repos.
2. **Update OpenAPI when API behavior changes**.
3. **No dead endpoints**: if route is removed, remove handler/docs/sdk usage together.
4. **Respect source-of-truth schema** (`sql.md`) for table/key/CF naming.
5. **Prefer simple structure**:
   - keep storage package layout as `client.go + repos.go` unless complexity requires split.

## Verification Checklist

Run after backend changes:

```bash
cd backend
make fmt
make vet
go test ./...
```

Storage package coverage targets:

```bash
go test ./internal/bigtable -coverprofile=/tmp/bt.cov && go tool cover -func=/tmp/bt.cov
go test ./internal/postgres -coverprofile=/tmp/pg.cov && go tool cover -func=/tmp/pg.cov
```

## Local Commands

### Backend

```bash
cd backend
make deps
make run
make bt-bootstrap
```

### SDK

```bash
cd sdk
uv sync --dev
uv run pytest tests/unit/
```

### Frontend

```bash
cd Frontend
npm install
npm run dev
```

## Environment Variables (Backend)

From `backend/.env.example`:

- `ENV`, `PORT`, `STORAGE_BACKEND`
- `BIGTABLE_PROJECT`, `BIGTABLE_INSTANCE`
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`
- `GCS_PROJECT`, `GCS_DERIVED_BUCKET`
- `PUBSUB_PROJECT`, `TOPIC_MCAP_FINALIZED`, `TOPIC_ASSET_EVENTS`
- `GRACE_TOKEN`
- `GOOGLE_APPLICATION_CREDENTIALS` (optional local)

## Definition of Done (Backend Changes)

- Code compiles and tests pass (`go test ./...`)
- Route/handler/usecase/repo are consistent
- OpenAPI updated if response/request/status changed
- README/CLAUDE guidance updated if architecture or workflow changed
