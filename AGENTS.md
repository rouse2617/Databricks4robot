# cyber-databrew

All project rules, workflows, skills, and MCP policies live under **`docs/agents/`**.

**Read these on every task:**

1. [`docs/agents/AI-RULES.md`](docs/agents/AI-RULES.md) — single source of truth (automatic behavior)
2. [`docs/agents/SETUP.md`](docs/agents/SETUP.md) — silent environment check
3. [`docs/agents/deploy-before-commit.md`](docs/agents/deploy-before-commit.md) — before any runtime `git commit` / `git push`
4. [`docs/agents/deploy-verification.md`](docs/agents/deploy-verification.md) — how to verify on dev after deploy (UI + API + regression)

Tool entry map (Cursor vs Codex vs others): [`docs/agents/TOOL-ENTRY.md`](docs/agents/TOOL-ENTRY.md)

Apply **Automatic behavior** without asking the user to "prepare environment" or "follow the workflow".
