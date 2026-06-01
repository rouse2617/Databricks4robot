# Decisions — CYB-1516

## 2026-06-01 — OpenSpec checkpoint approved
- **Context**: The user asked the agent to handle OpenSpec and start implementation after re-checking the live docs page.
- **Decision**: Proceed with implementation using this OpenSpec change as the approved checkpoint.
- **Alternatives**: Stop again for an explicit checkpoint phrase.
- **Rationale**: The current user instruction explicitly asks to update OpenSpec and begin work.

## 2026-06-01 — GitHub fetch unavailable
- **Context**: `git fetch origin dev` and GitHub HTTPS checks timed out from this machine, while the existing worktree and live site show newer docs navigation than the local `origin/dev` ref.
- **Decision**: Keep the CYB-1516 branch scoped to this issue and manually align only the affected docs navigation with the live state.
- **Alternatives**: Wait for GitHub connectivity before making progress.
- **Rationale**: The user asked to continue, and the targeted changes can be made without touching unrelated CYB-1515 docs content.

## 2026-06-01 — Local SEO verification split origin
- **Context**: The docs site is served locally under `http://127.0.0.1:3016/doc/`, while Lighthouse checks crawler metadata at the origin root. In production, the origin root is served by the main frontend.
- **Decision**: Verify docs metadata under `/doc/*` with the docs preview and root metadata under `/robots.txt` and `/llms.txt` with the frontend preview.
- **Alternatives**: Treat the docs preview Lighthouse SEO score as authoritative even though it cannot serve the production root frontend.
- **Rationale**: This matches the deployed split between the root app and `/doc/` docs site.

## 2026-06-01 — PR before Cloud Run dev deploy
- **Context**: The diff touches `Frontend/public/*` only to add root static crawler metadata, and the user explicitly asked to submit a PR to `dev`.
- **Decision**: Open the PR with local build, local preview, Chrome MCP, and static asset verification evidence; defer Cloud Run dev deploy verification to the PR workflow or a follow-up `/deploy-cloudrun-dev` run before merge.
- **Alternatives**: Block PR creation until a local Cloud Run image build, push, deploy, and user deploy approval are completed.
- **Rationale**: The changed frontend surface is static public files with local preview verification, and the user requested PR creation now. The deferred deploy verification remains visible in the PR.
