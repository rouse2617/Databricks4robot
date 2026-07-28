# Context files

## Repository rules and workflow

- `docs/agents/AI-RULES.md`
- `docs/agents/HARNESS.md`
- `docs/agents/spec-writing-skill.md`
- `docs/agents/deploy-before-commit.md`
- `docs/agents/deploy-verification.md`

## Backend pipeline and dependencies

- `backend/internal/transpiler/pipeline.go`
- `backend/internal/models/pipeline.go`
- `backend/internal/models/pipeline_component.go`
- `backend/internal/models/pipeline_config.go`
- `backend/internal/handlers/pipeline/`
- `backend/internal/handlers/pipeline_component/ingest_auth.go`
- `backend/internal/usecase/pipeline/`
- `backend/internal/repository/pipeline_repository.go`
- `backend/internal/postgres/pipeline_repo.go`
- `backend/internal/postgres/pipeline_component_repo.go`
- `backend/internal/config/config.go`
- `backend/routes/routes.go`
- `backend/migrations/`

## Frontend

- `Frontend/src/api/pipelineApi.ts`
- `Frontend/src/components/pipeline/types.ts`
- `Frontend/src/components/pipeline/DeployPanel.tsx`
- `Frontend/src/components/pipeline/VersionHistoryDrawer.tsx`
- `Frontend/src/features/pipeline-designer/PipelineDesignerCanvas.tsx`
- `Frontend/src/hooks/useAuth.ts`

## Contracts, tests, and deployment

- `api/openapi.yaml`
- `docs/review/api-guide.md`
- `sdk/src/cyber_databrew_sdk/`
- `sdk/tests/unit/`
- `scripts/api-guide-smoke.sh`
- `scripts/dev-backend-env.sh`
- `deploy/cloudrun/backend-dev.sh`
- `.github/workflows/deploy-prod.yml`
