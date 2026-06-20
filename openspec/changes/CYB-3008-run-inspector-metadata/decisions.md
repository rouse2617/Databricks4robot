# Decisions - CYB-3008

## 2026-06-20 - Placeholder Linear ID

- **Context**: `LINEAR_API_KEY` is not present in the environment, so the agent cannot create or verify a real Linear issue in this session.
- **Decision**: Use `CYB-3008` as a temporary Runtime OS trace id for the local OpenSpec change.
- **Alternatives**: Stop implementation until Linear credentials are available.
- **Rationale**: The active Runtime OS goal asks for continued engineering progress, and missing Linear credentials do not block in-repo planning or tests.

## 2026-06-20 - Continue without another checkpoint prompt

- **Context**: The user explicitly said: "ok，这个以后不要问我了，你直接执行".
- **Decision**: Create the OpenSpec artifacts, record scope, and continue implementation without stopping for another confirmation.
- **Alternatives**: Stop after the OpenSpec checkpoint and ask the user to approve CYB-3008.
- **Rationale**: The current explicit user instruction has higher precedence than the default checkpoint pause while preserving traceability.

## 2026-06-20 - Metadata failures are non-fatal

- **Context**: Inputs, outputs, and runtime metadata are useful context but should not block logs, DAG, events, terminal, or Pod diagnostics.
- **Decision**: Load metadata with isolated failure handling in the frontend hook.
- **Alternatives**: Fail the whole detail load when any metadata endpoint fails.
- **Rationale**: Run Inspector stability is more important than making secondary context all-or-nothing.
