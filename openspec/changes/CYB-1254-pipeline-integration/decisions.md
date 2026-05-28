## 2026-05-28 — Backend major bugfix task precedence
- **Context**: The repository workflow normally stops after OpenSpec artifacts before runtime edits, and migrations are off-limits without explicit approval.
- **Decision**: Proceeded with the user-requested implementation and commit flow in the current task, using the existing CYB-1254 change for traceability.
- **Alternatives**: Stop for a separate OpenSpec confirmation or create a new Linear-backed change.
- **Rationale**: The current user instruction explicitly requested implementation of all listed backend major bug fixes and included migration changes.

## 2026-05-28 — Dev deploy gate not run
- **Context**: Runtime files under backend, Frontend, and backend/migrations changed; the repository deploy gate normally requires dev deploy verification before commit.
- **Decision**: Completed local targeted tests, frontend build, and full backend build/test, then proceeded toward the requested local commit without dev deploy.
- **Alternatives**: Build and deploy backend/frontend images to Cloud Run dev, apply migrations to dev, run smoke and Chrome DevTools MCP, then ask for commit approval.
- **Rationale**: The task explicitly requested local fixes, local tests, and a commit in this worktree; no deploy credentials or dev environment target were provided in the task.
