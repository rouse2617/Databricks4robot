# Decisions — CYB-1344

## 2026-05-28 — Branch creation deferred
- **Context**: The current worktree is on `feat/pipeline-integration` and already contains unrelated dirty frontend files and review screenshots.
- **Decision**: Create only OpenSpec artifacts in the current worktree and defer the issue branch until after the OpenSpec checkpoint.
- **Alternatives**: Switch immediately to `fix/CYB-1344-workflow-dag-edges` from `origin/dev`.
- **Rationale**: Switching branches now would risk carrying or disturbing unrelated user-owned worktree changes. Runtime code changes will not start until the checkpoint is approved.

## 2026-05-28 — Branch base uses pipeline integration
- **Context**: `origin/dev` does not contain the workflow detail API handler or React Flow workflow DAG page that CYB-1344 fixes.
- **Decision**: Implement in a separate worktree on `fix/CYB-1344-workflow-dag-edges` based on the current `feat/pipeline-integration` HEAD.
- **Alternatives**: Branch from `origin/dev` and reimplement the entire workflow detail feature first.
- **Rationale**: CYB-1344 is a follow-up bug in the pipeline integration workflow detail surface; using the feature branch base keeps the change scoped to normalized DAG edges and avoids disturbing unrelated dirty files in the main worktree.

## 2026-05-28 — Verification blockers are pre-existing
- **Context**: Full frontend test and lint commands fail on files unrelated to CYB-1344, and SDK full ruff fails on `src/cyber_databrew_sdk/managers/search.py`.
- **Decision**: Record those failures and rely on focused frontend tests, frontend build, touched-file Biome check, backend full tests, SDK unit tests, and OpenAPI parsing for this change.
- **Alternatives**: Fix unrelated frontend and SDK quality issues in the same change.
- **Rationale**: The unrelated failures are outside the workflow DAG edge contract and expanding the patch would increase review risk.
