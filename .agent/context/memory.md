# Agent Context Memory

## Project

cyber-databrew — Go 1.25 + Gin + pgx (backend), React 19 + TS + Vite + Ant Design 5 (frontend), Python asset-sdk (httpx + Pydantic). PostgreSQL primary, Elasticsearch search, BigQuery lakehouse, GCS. Deploy: Cloud Run (us-central1), Docker multi-stage.

**Architecture**: handlers → usecase → repository → postgres/es

**Off-limits** (require explicit approval): `backend/internal/middleware/auth*`, `backend/internal/outbox/`, `backend/migrations/`, `schemas/pg-phase0.sql`, `.env*` / credential files.

## User (Rick)

Project lead / full-stack engineer. Git user: ruipeng.huang.
- **No ritual** — just do it silently, don't ask permission for workflow steps
- **Automation over manual** — scripts + CI gates, not human checklists
- **Terse** — diff speaks for itself, brief status only
- **Chinese OK**

## Workflow Rules

- **Trivial path**: ≤20-line single-file no-API changes → skip OpenSpec, lightweight proposal only
- **Deploy fast path**: pure logic/usecase/repo changes → `go test ./...` enough, skip docker; CSS/text-only skip deploy
- **current-work.md**: agent-maintained, update on CYB start and merge
- **Deploy**: tag images with git SHA + `cloudrun-dev-latest`, record revision + URL. Frontend changes → Chrome DevTools MCP mandatory.
- **Commit**: Conventional Commits (`feat:`, `fix:`, `hotfix:`, `docs:`). Run `git diff --stat` before commit. API handler changes → API contract sync in same commit.
- **stale-changes.sh**: detect merged-but-not-archived change dirs. Run at session start and before PR.
- **Source `scripts/dev-backend-env.sh`** for canonical dev API URL — never guess.
