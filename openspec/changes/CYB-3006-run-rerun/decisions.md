# Decisions - CYB-3006

## 2026-06-20 - Placeholder Linear ID
- **Context**: `LINEAR_API_KEY` is not present in the environment, so the agent cannot create or verify a real Linear issue in this session.
- **Decision**: Use `CYB-3006` as a temporary Runtime OS trace id for the local OpenSpec change.
- **Alternatives**: Stop implementation until Linear credentials are available.
- **Rationale**: The active Runtime OS goal asks for continued engineering progress, and missing Linear credentials do not block in-repo planning or tests.

## 2026-06-20 - Continue without another checkpoint prompt
- **Context**: The user explicitly said: "ok，这个以后不要问我了，你直接执行".
- **Decision**: Create the OpenSpec artifacts, record scope, and continue implementation without stopping for another confirmation.
- **Alternatives**: Stop after the OpenSpec checkpoint and ask the user to approve CYB-3006.
- **Rationale**: The current explicit user instruction has higher precedence than the default checkpoint pause while preserving traceability.

## 2026-06-20 - Add rerun as an additive endpoint
- **Context**: The legacy `/pipeline-runs/{id}/retry` already performs full rerun behavior, but product `/runs/{id}/retry` is runtime-level retry.
- **Decision**: Add `/runs/{id}/rerun` instead of changing `/runs/{id}/retry`.
- **Alternatives**: Change `/runs/{id}/retry` to full rerun, or keep full rerun only on the legacy endpoint.
- **Rationale**: This preserves existing runtime retry semantics and gives product clients an explicit full-rerun operation.
