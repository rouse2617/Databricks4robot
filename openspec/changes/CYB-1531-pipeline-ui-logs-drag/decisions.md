# Decisions — CYB-1531

## 2026-06-01 — Scope selection
- **Context**: Dev validation found several Pipeline UI issues after a successful 100-run Argo batch.
- **Decision**: Treat log visibility and component drag identity as P0 blockers; keep larger run grouping and execution-target UX as follow-up scope unless they are cheap polish.
- **Alternatives**: Redesign the entire Pipeline Run UI in one PR.
- **Rationale**: The two P0 issues can cause immediate operational failure or incorrect pipelines, while broader UX work needs a separate product pass.


## 2026-06-01 — OpenSpec approval
- **Context**: User approved the OpenSpec checkpoint in chat with "可以".
- **Decision**: Proceed to runtime code changes for CYB-1531.
- **Alternatives**: Wait for a more explicit approval phrase.
- **Rationale**: The approval directly followed the OpenSpec checkpoint request.

## 2026-06-01 — Compact recovery context file missing
- **Context**: After context recovery, `.agent/context/current-work.md` was required by repo rules but absent in this worktree.
- **Decision**: Continue from `docs/agents/AI-RULES.md`, git status, and the recovered turn summary; do not synthesize a replacement context file in this PR.
- **Alternatives**: Stop for user input or add a new context file.
- **Rationale**: The missing file did not block reconstructing the active branch, issue, scope, and verification requirements.

## 2026-06-01 — Dev Argo server env alignment
- **Context**: The Cloud Run backend env sourced `ARGO_BASE_URL`, but the current Go client reads `ARGO_SERVER_URL`. Dev workflow detail initially failed with `argo server URL is empty`, then with HTTPS against an HTTP Argo service.
- **Decision**: Deploy dev with temporary env overlay `ARGO_SERVER_URL=http://10.2.1.211:2746` using the existing VPC connector. Do not commit the temporary env file.
- **Alternatives**: Change application config fallback or deployment script in this PR.
- **Rationale**: The code fix scope is log/pod resolution and drag identity. The runtime env overlay restores dev verification without expanding the PR into deploy-config migration.

## 2026-06-01 — Log request filtering
- **Context**: After resolving node ids to pod names, Argo log requests returned 200 with empty logs because `grep=podName` filtered out normal log content that did not contain the pod name.
- **Decision**: Use `podName` only for pod selection and remove the `grep` query parameter.
- **Alternatives**: Grep on original node id or post-filter in the backend.
- **Rationale**: The endpoint should return pod logs for the selected node, not only lines containing an implementation-specific identifier.
