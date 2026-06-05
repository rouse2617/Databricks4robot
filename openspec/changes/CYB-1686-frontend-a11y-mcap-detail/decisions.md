# CYB-1686 Decisions

## 2026-06-05 - Continue after user approval

- **Context**: Runtime frontend changes normally stop after OpenSpec artifact creation for explicit approval.
- **Decision**: Continue implementation in the worktree after the user said "可以，在worktree 做".
- **Alternatives**: Stop after OpenSpec and ask for a second confirmation.
- **Rationale**: The current instruction explicitly approved doing the work in a worktree.

## 2026-06-05 - Full Vitest suite has unrelated failures

- **Context**: Focused tests for the touched stable surface pass, but `npm run test -- --run` fails in existing DeployPanel, PipelinePage, WorkflowDetailPage, WorkflowExecutionList, and SettingsPage tests.
- **Decision**: Record the full-suite failures and keep the focused verification evidence for this change.
- **Alternatives**: Expand this PR to repair the unrelated flaky/stale tests.
- **Rationale**: The failing assertions are outside the MCAP detail route and form identifier implementation. Several failures reflect existing behavior/test drift, such as DeployPanel expecting `?runId=` in a navigation path while the implementation navigates without it.

## 2026-06-05 - Dev browser smoke pending

- **Context**: Local dev server redirects unauthenticated browser sessions to `/login`, and this turn has not deployed frontend dev.
- **Decision**: Leave Chrome DevTools MCP smoke pending until the change is deployed or an authenticated local session is available.
- **Alternatives**: Mock auth in application code or bypass protected routing.
- **Rationale**: The route behavior is covered by component tests and build; bypassing auth in runtime code would be inappropriate for this fix.

## 2026-06-05 - PR follow-up for failed node labels

- **Context**: After PR creation, dev UI still showed Argo technical node names such as `workflow.step-step-1` in the failed-node summary, while the DAG cards displayed mapped business labels.
- **Decision**: Push a follow-up commit to the same PR that filters non-actionable DAG/Steps parent nodes and uses `displayName` for failed-node summaries and status sync warnings.
- **Alternatives**: Leave the PR unchanged and open a separate issue.
- **Rationale**: This is the same workflow-detail failure-diagnosis surface already covered by the PR, and the user asked about it before deployment verification.
