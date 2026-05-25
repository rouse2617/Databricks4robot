---
name: project-context
description: cyber-databrew tech stack, architecture, active CYB work, off-limits zones
metadata:
  type: project
---

# Project Context — cyber-databrew

## Stack

- **Backend**: Go 1.25, Gin, pgx/v5, BigQuery, Pub/Sub, GCS, Prometheus
- **Frontend**: React 19 + TypeScript + Vite, Ant Design 5, react-router-dom v7, Biome, Vitest
- **SDK**: Python asset-sdk (httpx + Pydantic v2), hatchling, pytest
- **Storage**: PostgreSQL (primary), Elasticsearch (search), BigQuery (lakehouse), GCS
- **Deploy**: Cloud Run (us-central1), Docker multi-stage, `deploy/cloudrun/*-dev.sh`

## Architecture

- Backend: handlers → usecase → repository → postgres/es (`cmd/server/main.go` → `routes/routes.go`)
- Frontend: pages → feature components → common components → hooks → api client
- SDK: thin httpx wrapper, Pydantic models

## Off-limits (require explicit approval)

- `backend/internal/middleware/auth*`
- `backend/internal/outbox/`
- `backend/migrations/`
- `schemas/pg-phase0.sql`
- `.env*` / credential files

## Active CYB issues

See `.agent/context/current-work.md` for live state.

**Why:** Quick recovery after context compaction — know what's in flight without reading all of dev branch history.

**How to apply:** Read `current-work.md` at session start. Update it when starting/finishing a CYB issue.
