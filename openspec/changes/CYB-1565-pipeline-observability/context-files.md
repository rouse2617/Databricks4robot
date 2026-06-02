# Context files — CYB-1565

openspec/changes/CYB-1564-run-events-watcher/design.md  # current run event ledger design
backend/internal/models/pipeline.go  # run/node/event models
backend/internal/usecase/pipeline/usecase.go  # watcher, run refresh, event derivation
backend/internal/postgres/pipeline_repo.go  # persistence patterns
backend/internal/handlers/pipeline/handler.go  # pipeline API handlers
Frontend/src/pages/WorkflowDetailPage.tsx  # run detail UI
Frontend/src/pages/useWorkflowDetail.ts  # detail data loading
Frontend/src/api/pipelineApi.ts  # pipeline API client
api/openapi.yaml  # public contract
docs/review/api-guide.md  # API guide
sdk/src/cyber_databrew_sdk/managers/pipelines.py  # SDK pipeline methods
