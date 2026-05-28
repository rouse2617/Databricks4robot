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

## 2026-05-28 — Workflow operations stop endpoint
- **Context**: The requested Argo workflow operations map includes Stop, and the backend Argo client already implements `StopWorkflow`, but the public workflow monitoring routes did not expose `/workflows/:name/stop`.
- **Decision**: Added `POST /api/v1/workflows/{name}/stop` with OpenAPI, api-guide, smoke script, and route/handler test coverage so the UI does not alias Stop to Terminate.
- **Alternatives**: Hide Stop in the UI or call Terminate from the Stop button.
- **Rationale**: Stop and Terminate are distinct Argo operations; exposing the existing client method preserves operator intent.

## 2026-05-28 — Dev deploy gate not run for workflow operations
- **Context**: Frontend and workflow API files changed; the repository deploy gate normally requires frontend dev deploy and Chrome DevTools MCP verification before commit.
- **Decision**: Completed local lint/build and targeted backend route tests, but did not deploy to dev before the requested local commit.
- **Alternatives**: Deploy frontend/backend dev and run browser verification before committing.
- **Rationale**: The task explicitly requested implementation and a local commit; no dev deploy credentials or target approval were provided in the task.
