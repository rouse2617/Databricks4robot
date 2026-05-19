# AI tool entry points (Cursor / Codex / others)

**Audience:** maintainers and agents. End-user onboarding is in the root [README](../../README.md) — it does not list IDE-specific paths.

All tools load the **same** rules under `docs/agents/`. There is **no** Cursor-only or Codex-only workflow for OpenSpec, deploy-before-commit, commit hooks, or PR checks.

---

## Where each tool starts

| Tool | How rules load (user-facing) | First file read |
|------|------------------------------|-----------------|
| **Cursor** | Open repo root; IDE loads project rules automatically | [`AI-RULES.md`](AI-RULES.md) via `.cursor/rules/00-ai-entry.mdc` |
| **Codex** | Open repo root | [`AGENTS.md`](../../AGENTS.md) → `docs/agents/AI-RULES.md` |
| **Claude Code** | Open repo root | [`CLAUDE.md`](../../CLAUDE.md) → `docs/agents/AI-RULES.md` |

### Maintainer: pointer files (do not document in README)

| Tool | Pointer in repo | Must contain |
|------|-----------------|--------------|
| Cursor | [`.cursor/rules/00-ai-entry.mdc`](../../.cursor/rules/00-ai-entry.mdc) | Link to `docs/agents/AI-RULES.md` only |
| Cursor | [`.cursor/rules/deploy-before-commit.mdc`](../../.cursor/rules/deploy-before-commit.mdc) | Link to `deploy-before-commit.md` only |
| Codex | [`AGENTS.md`](../../AGENTS.md) | Short pointer + link to `AI-RULES.md` |
| Claude Code | [`CLAUDE.md`](../../CLAUDE.md) | Short pointer + link to `AI-RULES.md` |
| Legacy | [`.cursorrules`](../../.cursorrules) | One-line pointer (if present) |

**Rule:** pointer files must not duplicate policy text. Change behavior in `docs/agents/`; update pointers only when paths move.

---

## Read order (every agent, every task)

| Step | Document | Purpose |
|------|----------|---------|
| 1 | [`AI-RULES.md`](AI-RULES.md) | Automatic behavior, precedence, verification tiers, decision log |
| 2 | [`SETUP.md`](SETUP.md) | Silent env check (Linear, skills, hooks, build smoke) |
| 3 | [`WORKFLOWS.md`](WORKFLOWS.md) | Bug / Feature / Hotfix paths |
| 4+ | As needed (table below) | Deploy, OpenSpec, skills, MCP |

---

## Full `docs/agents/` index

| Document | When to read |
|----------|----------------|
| [`AI-RULES.md`](AI-RULES.md) | Every task (mandatory) |
| [`SETUP.md`](SETUP.md) | First task in session / env failure |
| [`WORKFLOWS.md`](WORKFLOWS.md) | Classifying bug vs feature vs hotfix |
| [`spec-driven-workflow.md`](spec-driven-workflow.md) | Creating or archiving OpenSpec changes |
| [`spec-writing-skill.md`](spec-writing-skill.md) | Creating OpenSpec change artifacts (proposal, delta, design, tasks) |
| [`deploy-before-commit.md`](deploy-before-commit.md) | Before `git commit` / `git push` on runtime code |
| [`deploy-verification.md`](deploy-verification.md) | After Cloud Run dev deploy; §6 fixed regression lists |
| [`SKILLS.md`](SKILLS.md) | Which repo skills to trigger |
| [`MCP-POLICY.md`](MCP-POLICY.md) | Before any MCP call (schema paths + Linear) |
| [`TOOL-ENTRY.md`](TOOL-ENTRY.md) | Maintainer: adding a new IDE or pointer file |

---

## Same obligations for every tool

| Requirement | Document |
|-------------|----------|
| Linear + OpenSpec before runtime code | `AI-RULES.md`, `WORKFLOWS.md`, `spec-driven-workflow.md` |
| Rule conflicts / off-limits / hotfix | `AI-RULES.md` → Rule precedence |
| Verification S/M/L (not always full build) | `AI-RULES.md` → Verification tiers |
| Agent decision audit | `openspec/changes/.../decisions.md` or PR **Agent decisions** |
| Deploy verify before commit/push (runtime) | `deploy-before-commit.md`, `deploy-verification.md` |
| Conventional Commits | `AI-RULES.md`, README Commit 规范, `.githooks/commit-msg` |
| PR template + CI gates | `.github/pull_request_template.md`, `.github/workflows/` |

Do not tell users that deploy, OpenSpec, or hooks apply only when using Cursor.

---

## Adding a new AI tool

1. Add a root pointer file (e.g. `WARP.md`) with 3–5 lines: link to [`AI-RULES.md`](AI-RULES.md) and [`SETUP.md`](SETUP.md).
2. Add a row to the table **Where each tool starts** above.
3. Do **not** copy workflow text into the pointer.
4. Update this file in the same PR.
