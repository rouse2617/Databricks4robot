# Context Files — CYB-1569

- `Frontend/src/components/pipeline/WorkflowNodeDetailPanel.tsx` — current runtime tab and terminal placeholder.
- `Frontend/src/api/workflowApi.ts` — workflow detail, logs, and pod diagnostics client surface.
- `backend/internal/handlers/workflow/handler.go` — workflow detail/logs/pod diagnostics handlers.
- `backend/internal/k8s/` — Kubernetes client and Pod diagnostics access path.
- `backend/internal/usecase/pipeline/usecase.go` — run lookup, execution targets, run events.
- `backend/internal/models/pipeline.go` — execution target, run event, and run/node models.
- `backend/internal/postgres/pipeline_repo.go` — execution target/run/run event repositories.
- `backend/internal/postgres/audit_sink.go` — audit event persistence.
- `api/openapi.yaml` — API contract source.
- `docs/review/api-guide.md` — operator API examples.
- `docs/review/round2-design-brief.md` — prior UX notes mentioning xterm/terminal considerations.
