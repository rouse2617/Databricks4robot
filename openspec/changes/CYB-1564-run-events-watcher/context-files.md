# Context files — CYB-1564

docs/agents/AI-RULES.md  # workflow, API contract sync, off-limits migration policy
docs/agents/deploy-before-commit.md  # runtime deploy gate
docs/agents/deploy-verification.md  # dev verification and MCP requirements
backend/internal/models/pipeline.go  # existing run and node models
backend/internal/repository/pipeline_repository.go  # run/node repository contracts
backend/internal/postgres/pipeline_repo.go  # pipeline run persistence patterns
backend/internal/usecase/pipeline/usecase.go  # run lifecycle, Argo sync, node extraction
backend/internal/handlers/pipeline/handler.go  # pipeline run API patterns
backend/routes/routes.go  # route registration
backend/cmd/server/core.go  # repository/usecase wiring
backend/internal/handlers/asset/handler.go  # existing event list/SSE response patterns
api/openapi.yaml  # public API contract
docs/review/api-guide.md  # public API guide
sdk/src/cyber_databrew_sdk/client.py  # SDK surface wiring
Frontend/src/api/pipelineApi.ts  # pipeline API client
Frontend/src/pages/useWorkflowDetail.ts  # workflow detail data hook
Frontend/src/pages/WorkflowDetailPage.tsx  # run events UI placeholder and detail page
