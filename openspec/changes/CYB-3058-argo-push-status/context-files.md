# Context files — CYB-3058

Read before implementing:

backend/internal/transpiler/transpiler.go            # Options (38-52), WorkflowSpec skeleton (80-107) — inject exit hook + HTTP template
backend/internal/usecase/pipeline/usecase.go         # Deploy (2940), Options assembly (3092-3103), appendRunEvent (1278), applyWorkflowToRun (1813), persistRunObservation (1949, guard here), isActiveDeploymentStatus (5067), StartRunEventWatcher (2727), SyncActiveRunEvents (2506), TTL setter/getter pattern (225-233), syncBackfillItemStatusFromRun (1999)
backend/internal/postgres/pipeline_repo.go           # PipelineRunEventRepo.Append idempotent ON CONFLICT (1468-1511), FindByWorkflowName (1181)
backend/internal/repository/pipeline_repository.go   # FindByWorkflowName interface (105) — update test mocks in same commit
backend/internal/middleware/auth.go                  # StaticTokenAuth (33) pattern for new ArgoWebhookAuth
backend/internal/handlers/pipeline_component/ingest_auth.go  # ReleaseIngestAuth (17) + constantTimeTokenEqual (80) — closest machine-token auth pattern
backend/internal/handlers/algorun/handler.go         # handler struct/New pattern (19-25) for HandleRunWebhook
backend/routes/routes.go                             # api group (213), algo-runs finish (351), releaseIngest independent group (198) — register webhook here
backend/internal/config/config.go                    # Argo env block (137-139, 238-242) — add webhook URL/token + watcher interval/scan-limit
backend/cmd/server/core.go                           # StartRunEventWatcher call (147, hardcoded 3s/100), pipeline usecase New (115), SetObservabilityRepositories (130)
api/openapi.yaml                                     # add webhook path + schema
docs/review/api-guide.md                             # add webhook curl example (happy + error)
scripts/dev-backend-env.sh                           # source for dev smoke
docs/agents/deploy-verification.md                   # dev verification after deploy
