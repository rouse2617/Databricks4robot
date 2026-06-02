# Decisions — CYB-1564

## 2026-06-03 — Migration approval required
- **Context**: P0.1 needs a durable `pipeline_run_events` table.
- **Decision**: Treat `backend/migrations/` as off-limits and stop after OpenSpec until the user approves continuing with migration/runtime implementation.
- **Alternatives**: Store events only in memory or reuse an existing table.
- **Rationale**: In-memory storage does not satisfy durable history, and reusing asset events would blur asset audit with workflow execution history.

## 2026-06-03 — Migration approval granted
- **Context**: The user confirmed "然后开始编码，在新的workttree 里面" after the OpenSpec checkpoint explicitly asked for migration approval.
- **Decision**: Proceed with `backend/migrations/046_pipeline_run_events.sql` in the isolated `/Users/rick/cyber-databrew-cyb1564` worktree.
- **Alternatives**: Defer persistence and implement only frontend placeholders.
- **Rationale**: Durable run history, audit, and watcher deduplication require a real table and indexes.

## 2026-06-03 — Push before dev deploy
- **Context**: Runtime changes normally require dev deploy verification before commit/push, but the user explicitly requested "可以push 分支先".
- **Decision**: Commit and push the feature branch before dev deployment.
- **Alternatives**: Stop and deploy dev first.
- **Rationale**: The user's current instruction is to make the branch available first; local Tier L verification and pre-commit have already passed. Dev migration/deploy verification remains pending before merge.
