# Context files — CYB-1123

backend/internal/handlers/algorun/handler.go  # algo-runs list request/response contract
backend/internal/postgres/algo_runs.go  # repo pagination semantics
Frontend/src/api/algoRuns.ts  # frontend request params and response typing
Frontend/src/pages/AlgoRunsPage.tsx  # UI pagination wiring
backend/internal/handlers/delivery/handler.go  # C2 draft/retry/commit/ack behavior
backend/internal/postgres/repos.go  # delivery index mutation semantics
backend/internal/models/delivery_transition.go  # delivery state model
api/openapi.yaml  # API contract sync
docs/review/api-guide.md  # curl and behavior docs
docs/agents/deploy-verification.md  # deploy verification and regression scope
