## 2026-06-05 — Rebase onto latest dev before deploy
- **Context**: The worktree started from `f2f749d`, but `origin/dev` advanced to `d7e2c6d` before deploy verification.
- **Decision**: Rebased the worktree onto latest `origin/dev` and resolved the two frontend conflicts in `WorkflowExecutionList.tsx` and `WorkflowExecutionList.test.tsx` before continuing.
- **Alternatives**: Deploy from the older base and resolve drift later.
- **Rationale**: This feature changes both backend API fields and frontend execution-list rendering. Deploying from a stale base would risk validating the wrong integration surface.

## 2026-06-05 — Frontend lint baseline noise kept out of scope
- **Context**: `cd Frontend && npm run lint` fails on historical repository issues outside this change:
  - `src/components/asset-detail/ActionsTimelineTab.tsx` (`useExhaustiveDependencies`)
  - formatting in `src/components/pipeline/NodeConfigPanel.tsx`
  - formatting in `src/pages/ComponentManager.tsx`
  - formatting in `src/pages/LoginPage.tsx`
- **Decision**: Fixed lint/format issues in touched CYB-1684 files and kept the unrelated baseline failures out of scope for this branch.
- **Alternatives**: Sweep and reformat the unrelated files in the same branch.
- **Rationale**: CYB-1684 is about execution traceability. Pulling unrelated baseline cleanup into the same PR would increase review noise and deployment risk without improving the feature itself.

## 2026-06-05 — Skip frontend deploy for this branch by user instruction
- **Context**: This change touches `Frontend/`, but the user explicitly instructed `前端不用构建了` while I was in the deploy phase.
- **Decision**: Deployed backend dev only and skipped frontend image build/deploy for this branch. Kept browser/dev UI verification as a follow-up item rather than blocking PR creation.
- **Alternatives**: Ignore the user instruction and continue the frontend Cloud Run deploy + browser verification gate.
- **Rationale**: The user prioritized PR momentum over frontend dev deploy for this branch. I preserved that instruction and recorded the verification gap explicitly instead of silently claiming full deploy coverage.
