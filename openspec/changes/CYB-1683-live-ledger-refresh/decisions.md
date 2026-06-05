# Decisions — CYB-1683

## 2026-06-05 — Browser MCP fallback

- **Context**: This frontend change requires browser verification. Local frontend was started on `http://127.0.0.1:5188/` against the Cloud Run dev backend.
- **Decision**: Chrome DevTools MCP was attempted but failed with `Transport closed`; agentyc browser failed with `All connection attempts failed`. Used local Playwright smoke as fallback.
- **Alternatives**: Full MCP-driven workflow detail verification after a dev deploy or with a valid local dev access token.
- **Rationale**: The automated unit tests and production build validate the code path; local browser smoke reached the login page without request failures beyond expected unauthenticated 401s, but authenticated detail-page UI verification remains blocked without a working browser MCP/session.

## 2026-06-05 — Authenticated local smoke

- **Context**: User provided `ruipeng.huang@cyberorigin.ai` for local dev login.
- **Decision**: Ran local frontend on `http://127.0.0.1:5188/` with Cloud Run dev backend, logged in via email, opened `/pipeline?tab=executions`, clicked the first `查看`, and verified the detail route included `runId`.
- **Alternatives**: Cloud Run frontend dev deploy verification; deferred because the user previously asked not to deploy from this worktree.
- **Rationale**: The smoke verified the exact regression: detail page loaded node detail, displayed `Succeeded`, did not show `等待资源快照`, and no longer issued `/api/v1/workflows/{runId}` 404 requests. Screenshot: `local-workflow-detail-final.png`.

## 2026-06-05 — PR before dev deploy

- **Context**: Runtime frontend changes normally require Cloud Run dev deploy verification before commit. User explicitly asked to submit the PR now.
- **Decision**: Open PR with local authenticated smoke, targeted tests, Biome, build, and pre-commit evidence. Mark Cloud Run frontend dev deploy verification as pending in the PR.
- **Alternatives**: Build and deploy frontend dev image before committing.
- **Rationale**: User controls the next deployment step for this iteration, and the local smoke exercised the same Cloud Run dev backend data path without changing deployed services.
