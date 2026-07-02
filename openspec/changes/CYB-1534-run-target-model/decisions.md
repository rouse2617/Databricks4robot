# Decisions — CYB-1534 Run Target Model

## 2026-06-02 — Migration approval
- **Context**: First-class run and target persistence requires new Postgres tables.
- **Decision**: The user approved editing `backend/migrations/` for this follow-up.
- **Alternatives**: Continue using compatibility mapping over `pipeline_deployments`.
- **Rationale**: Durable asset x node status, multi-target execution, and run history cannot be modeled cleanly without persistence.

## 2026-06-02 — Frontend consumption deferred
- **Context**: CYB-1534 exposes new `/api/v1/pipeline-runs` endpoints and keeps legacy `/deployments` compatibility.
- **Decision**: Sync OpenAPI, api-guide, smoke, and SDK in this backend PR; defer frontend API consumption to a UI PR because the current frontend continues to use compatible deployment endpoints.
- **Alternatives**: Update `Frontend/src/api/pipelineApi.ts` in the same branch.
- **Rationale**: Avoid mixing backend schema migration work with UI routing changes while preserving current UX behavior.
