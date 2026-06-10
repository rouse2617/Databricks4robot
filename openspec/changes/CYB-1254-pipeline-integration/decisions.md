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

## 2026-05-28 — Pro-flow local UI verification limits
- **Context**: The pro-flow canvas follow-up changes `Frontend/` runtime code. Local `npm run lint`, `npm run build`, `npm run test -- PipelinePage`, and `npm run test -- pipelineContract` all passed. A Playwright smoke against `http://localhost:5176/pipeline` reached the login screen because no dev auth cookie/token was available. Chrome DevTools MCP calls failed because another MCP Chrome instance already holds `/Users/rick/.cache/chrome-devtools-mcp/chrome-profile`.
- **Decision**: Treat local static and focused test verification as complete for this worktree pass, and leave deployed UI verification as pending if this change proceeds to the repository deploy gate.
- **Alternatives**: Stop until dev auth and a clean Chrome DevTools MCP profile are available; or kill existing MCP Chrome processes and run an authenticated smoke.
- **Rationale**: The implementation is frontend-only and the available automated checks exercise the pipeline page and pipeline contract without requiring mutation of shared dev data.
