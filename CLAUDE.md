# cyber-databrew

All project rules, workflows, skills, and MCP policies live under **`docs/agents/`**.

**Read these on every task:**

1. [`docs/agents/AI-RULES.md`](docs/agents/AI-RULES.md) — single source of truth (automatic behavior)
2. [`docs/agents/SETUP.md`](docs/agents/SETUP.md) — silent environment check
3. [`docs/agents/deploy-before-commit.md`](docs/agents/deploy-before-commit.md) — before any runtime `git commit` / `git push`
4. [`docs/agents/deploy-verification.md`](docs/agents/deploy-verification.md) — dev verification after deploy

Tool entry map: [`docs/agents/TOOL-ENTRY.md`](docs/agents/TOOL-ENTRY.md)

Apply **Automatic behavior** without asking the user to "prepare environment" or "follow the workflow".

---

## Compact recovery (auto-triggered after context compaction)

After context compaction, you MUST re-read `docs/agents/AI-RULES.md` before continuing any development work. Key rules that MUST NOT be forgotten:

1. **API contract first** — define response shape in OpenAPI + TS types BEFORE writing handler code
2. **Test mock sync** — when adding interface methods, update ALL test mocks in the SAME commit
3. **Migration check** — apply `scripts/apply-migration-dev.sh` BEFORE deploying backend with new migrations
4. **Deploy before commit** — build → push → deploy → verify → then commit (not the other way around)
5. **OpenSpec checkpoint** — stop after proposal+tasks, ask user to confirm before coding
