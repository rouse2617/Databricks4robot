# cyber-databrew

All project rules, workflows, skills, and MCP policies live under **`docs/agents/`**.

**Read these on every task:**

1. [`docs/agents/AI-RULES.md`](docs/agents/AI-RULES.md) — single source of truth (automatic behavior)
2. [`docs/agents/HARNESS.md`](docs/agents/HARNESS.md) — portable runtime guardrails (shared scripts, adapter rules)
3. [`.agent/context/current-work.md`](.agent/context/current-work.md) — active CYB / branch / constraints (compact recovery)
4. [`docs/agents/SETUP.md`](docs/agents/SETUP.md) — silent environment check
5. [`docs/agents/deploy-before-commit.md`](docs/agents/deploy-before-commit.md) — before any runtime `git commit` / `git push`
6. [`docs/agents/deploy-verification.md`](docs/agents/deploy-verification.md) — dev verification after deploy

Tool entry map: [`docs/agents/TOOL-ENTRY.md`](docs/agents/TOOL-ENTRY.md)

Apply **Automatic behavior** without asking the user to "prepare environment" or "follow the workflow".

**Compact recovery:** After context compaction, re-read `docs/agents/AI-RULES.md` and `.agent/context/current-work.md` before continuing development work.
