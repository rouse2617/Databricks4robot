# Decisions - CYB-3005

## 2026-06-20 - Placeholder Linear ID
- **Context**: `LINEAR_API_KEY` is not present in the environment, so the agent cannot create or verify a real Linear issue in this session.
- **Decision**: Use `CYB-3005` as a temporary Runtime OS trace id for the local OpenSpec change.
- **Alternatives**: Stop implementation until Linear credentials are available.
- **Rationale**: The active Runtime OS goal asks for continued engineering progress, and missing Linear credentials do not block in-repo planning or tests.

## 2026-06-20 - Continue without another checkpoint prompt
- **Context**: The user explicitly said: "ok，这个以后不要问我了，你直接执行".
- **Decision**: Create the OpenSpec artifacts, record scope, and continue implementation without stopping for another confirmation.
- **Alternatives**: Stop after the OpenSpec checkpoint and ask the user to approve CYB-3005.
- **Rationale**: The current explicit user instruction has higher precedence than the default checkpoint pause while preserving traceability.

## 2026-06-20 - Use PipelineJSON for this projection slice
- **Context**: Runtime OS eventually wants first-class RunInput storage, but adding a table would require a migration and deployed database coordination.
- **Decision**: Store sanitized config input metadata in `PipelineJSON._run_config_inputs` for now.
- **Alternatives**: Add a dedicated `run_inputs` table in this slice.
- **Rationale**: This keeps the scope reversible and allows `/runs/{id}/inputs` to expose the correct product model without schema risk.
