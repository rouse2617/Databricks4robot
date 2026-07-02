# Context files — CYB-1609

backend/internal/argo/client.go # Argo log options and stream methods
backend/internal/argo/logs.go # bounded log parsing and truncation behavior
backend/internal/handlers/workflow/handler.go # JSON log handler and query parsing
backend/internal/handlers/workflow/logs_sse.go # SSE log handler and stream lifecycle
backend/internal/handlers/workflow/handler_test.go # existing workflow log tests
backend/routes/routes.go # workflow route registration
api/openapi.yaml # workflow log contract
docs/review/api-guide.md # user-facing curl examples
Frontend/src/api/workflowApi.ts # frontend workflow log client
Frontend/src/pages/useWorkflowDetail.ts # log hook state and follow/download behavior
Frontend/src/pages/WorkflowDetailPage.tsx # log viewer UI
Frontend/src/pages/WorkflowDagNode.tsx # failed-node log action
Frontend/src/pages/useWorkflowDetail.test.tsx # frontend log hook tests
docs/agents/deploy-verification.md # frontend MCP and dev verification requirements
