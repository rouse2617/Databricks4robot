# Decisions - CYB-3010

## 2026-06-20 - Placeholder Linear ID
- **Context**: `LINEAR_API_KEY` is not present in the environment, so the agent cannot create or verify a real Linear issue in this session.
- **Decision**: Use `CYB-3010` as the local Runtime OS trace id because the user explicitly named this hardening slice as CYB-3010.
- **Alternatives**: Stop implementation until Linear credentials are available.
- **Rationale**: The active Runtime OS goal requires continued engineering progress, and missing Linear credentials do not block in-repo OpenSpec, tests, or dev verification.

## 2026-06-20 - Continue without another checkpoint prompt
- **Context**: Repository rules normally stop after OpenSpec artifacts, but the user previously said "ok，这个以后不要问我了，你直接执行" and now explicitly asked to continue this hardening work.
- **Decision**: Create the OpenSpec artifacts, record the scope, and continue implementation without another approval interruption.
- **Alternatives**: Stop after the OpenSpec checkpoint and ask the user to approve CYB-3010.
- **Rationale**: The current user instruction has higher precedence than the default checkpoint pause while preserving traceability in this change directory.

## 2026-06-20 - Schema migration approval
- **Context**: `backend/migrations/` is normally off-limits and requires explicit approval, but the user explicitly requested persistent `run_relations` and `run_inputs` tables in this slice.
- **Decision**: Add a migration for durable Run relation and Run input facts as part of CYB-3010.
- **Alternatives**: Keep using `PipelineJSON` and RunEvent projection only.
- **Rationale**: The user identified these projections as the current weak point; making them durable is the purpose of this hardening pass.

## 2026-06-20 - Leave RunOutput and Artifact persistence for the next slice
- **Context**: The user listed RunOutput/Artifact persistence as P0, but combining outputs, artifacts, relations, inputs, parent Runs, and state projection in one migration would make the deploy and rollback surface too large.
- **Decision**: This slice implements parent Run, durable `run_relations`, durable `run_inputs`, diagnostics fallback, and batch aggregate health. RunOutput/Artifact persistence remains a follow-up slice.
- **Alternatives**: Add all new Run fact tables in one migration.
- **Rationale**: Relations and inputs are the Run Tree foundation. Outputs/artifacts can be layered after the core Run lineage and input facts are stable.
