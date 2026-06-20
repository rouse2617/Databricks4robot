# Decisions - CYB-3004

## 2026-06-20 - Placeholder Linear ID
- **Context**: `LINEAR_API_KEY` is not present in the environment, so the agent cannot create or verify a real Linear issue in this session.
- **Decision**: Use `CYB-3004` as a temporary Runtime OS trace id for the local OpenSpec change.
- **Alternatives**: Stop implementation until Linear credentials are available.
- **Rationale**: The active Runtime OS goal asks for continued engineering progress, and missing Linear credentials do not block in-repo planning or tests.

## 2026-06-20 - Work on local dev and defer push
- **Context**: The user explicitly said not to keep opening PRs during this goal and to work on `dev`, then push once after the full goal is complete.
- **Decision**: Continue on the local `dev` worktree, do not open PRs for individual slices, and do not push until the full Runtime OS goal is complete and verified.
- **Alternatives**: Keep creating a PR for each completed slice.
- **Rationale**: The current user instruction has precedence over the default PR cadence while preserving tests and deploy gates before the final push.

## 2026-06-20 - OpenSpec checkpoint retained
- **Context**: Repository rules require creating OpenSpec artifacts and stopping before application-code edits for runtime changes.
- **Decision**: Create the CYB-3004 OpenSpec artifacts and wait for user confirmation before editing additional `backend/`, `Frontend/`, `sdk/`, or `dagster/` runtime code.
- **Alternatives**: Treat the broad autonomous Runtime OS goal as approval for all future runtime slices.
- **Rationale**: This is a new runtime slice after CYB-3003, and the checkpoint keeps scope bounded before submit-path code changes.

## 2026-06-20 - User approved future OpenSpec checkpoints
- **Context**: After the CYB-3004 checkpoint, the user said: "ok，这个以后不要问我了，你直接执行".
- **Decision**: Continue future Runtime OS slices after writing OpenSpec artifacts without stopping for per-slice confirmation, while still recording scope, decisions, and verification in the change directories.
- **Alternatives**: Stop at each future OpenSpec checkpoint and wait for another explicit chat approval.
- **Rationale**: The current explicit user instruction has higher precedence than the default checkpoint pause, and the in-repo artifacts still preserve review traceability.
