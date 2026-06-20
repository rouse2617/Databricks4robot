# Decisions - CYB-3009

## 2026-06-20 - Placeholder Linear ID

- **Context**: `LINEAR_API_KEY` is not present in the environment, so the agent cannot create or verify a real Linear issue in this session.
- **Decision**: Use `CYB-3009` as a temporary Runtime OS trace id for the local OpenSpec change.
- **Alternatives**: Stop implementation until Linear credentials are available.
- **Rationale**: The active Runtime OS goal asks for continued engineering progress, and missing Linear credentials do not block in-repo planning or tests.

## 2026-06-20 - Continue without another checkpoint prompt

- **Context**: The user explicitly said: "ok，这个以后不要问我了，你直接执行".
- **Decision**: Create the OpenSpec artifacts, record scope, and continue implementation without stopping for another confirmation.
- **Alternatives**: Stop after the OpenSpec checkpoint and ask the user to approve CYB-3009.
- **Rationale**: The current explicit user instruction has higher precedence than the default checkpoint pause while preserving traceability.

## 2026-06-20 - Use RunEvent projection before a relation table

- **Context**: The current goal avoids risky schema work while still needing Run Tree visibility.
- **Decision**: Project relation rows from source Run events instead of creating a new `run_relations` table now.
- **Alternatives**: Add a migration and persistent relation repository immediately.
- **Rationale**: Event projection is additive, testable, deployable without migration risk, and matches the existing Run ledger direction.
