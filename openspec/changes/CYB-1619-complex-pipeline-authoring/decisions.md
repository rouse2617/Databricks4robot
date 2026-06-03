# Decisions — CYB-1619

## 2026-06-03 — Start from latest dev

- **Context**: User asked to implement component input/output UI, fan-in join authoring, and standard complex pipeline examples in a worktree based on latest dev.
- **Decision**: Created CYB-1619 and branch `feat/CYB-1619-complex-pipeline-authoring` from `origin/dev@6863897`.
- **Alternatives**: Continue from the main checkout, which currently has older branch state and untracked files.
- **Rationale**: A clean worktree avoids mixing this feature with previous local state.
