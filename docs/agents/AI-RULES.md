# AI Agent Rules — cyber-databrew

This is the **single source of truth** for all AI agents working in this repository (Cursor, Codex, Claude Code, and others). Tool-specific entry files only **point here** — see [`TOOL-ENTRY.md`](TOOL-ENTRY.md).

**Users do not run a separate "setup" or "compliance" step.** When someone describes a task in natural language, you MUST apply these rules automatically — without asking "should I follow the project workflow?" **Never imply that deploy, OpenSpec, or hooks are Cursor-only.**

---

## Automatic behavior (every task)

On **every** user message that involves this repo, do the following **before** editing code:

1. **Silent environment check** — Follow `docs/agents/SETUP.md` in the background. Fix what you can (deps, skills, **repo-local commit-msg hook** via `git config core.hooksPath .githooks`). Only interrupt the user if a required credential is missing (e.g. `LINEAR_API_KEY`). OpenSpec is **in-repo Markdown only** — never ask to install OpenSpec CLI.
2. **Classify the task** — Bug fix, feature, hotfix, or docs/infra-only. Pick the path in `docs/agents/WORKFLOWS.md`.
3. **Linear** — Ensure a Linear Issue exists (`CYB-xxx` in this workspace). Create or link via MCP if the user did not provide one.
4. **OpenSpec** — For any runtime change, create or use `openspec/changes/CYB-{id}-{slug}/` **before** writing application code. Never skip because the user did not mention OpenSpec. Follow [`spec-writing-skill.md`](spec-writing-skill.md) for artifact quality (proposal, delta, design, tasks, `context-files.md`, decisions). Read `context-files.md` (or `## Context files` in `tasks.md`) before implementation.
5. **Branch** — Use `fix/CYB-{id}-*`, `feat/CYB-{id}-*`, or `hotfix/CYB-{id}-*` as appropriate (`DAT-*` accepted by CI for legacy).
6. **After each code change** — Run verification at the **tier** matching diff scope (see [Verification tiers](#verification-tiers)); log non-obvious choices in `decisions.md` when required. **Bugs:** follow [`systematic-debugging`](skills/systematic-debugging/SKILL.md) before speculative fixes.
7. **Before commit/push** — Follow [`deploy-before-commit.md`](deploy-before-commit.md) and [`deploy-verification.md`](deploy-verification.md) **§6** with [`verification-before-completion`](skills/verification-before-completion/SKILL.md) (evidence before claims). **If the diff touches `Frontend/`:** deploy frontend dev → Agent **must** run **Chrome DevTools MCP** (tasks path + screenshot + console). **Backend/sdk-only:** API smoke only — no browser MCP.
8. **PR** — Fill `.github/pull_request_template.md` completely when opening a PR.

Do **not** ask the user to confirm that you will follow this workflow. Do **not** wait for them to say "按规范" or "prepare environment".

---

## What users typically say (you handle the rest)

| User says (examples) | You automatically do |
|----------------------|----------------------|
| "BatchDelete 删完不刷新" | Infer bug → Linear issue → OpenSpec change → `fix/CYB-*` branch → fix → verify → PR fields |
| "做 CYB-58 导出 API" | Load CYB-58 → feature path → OpenSpec (incl. design if feature) → `feat/CYB-58-*` → implement |
| "hotfix 线上 P0" | `hotfix-approved` label → skip OpenSpec gate only → still deploy verify → note backfill spec T+2 |
| "只改 README" | Docs-only path → no OpenSpec for runtime |

---

## Workflow constraints (mandatory)

- Runtime paths: `backend/`, `Frontend/`, `sdk/`, `dagster/` — see `docs/agents/spec-driven-workflow.md`
- Commit format: Conventional Commits — `type(scope): description`
- Full step tables: `docs/agents/WORKFLOWS.md`

## Off-limits zones (require explicit approval to touch)

- `backend/internal/middleware/auth*`
- `backend/internal/outbox/`
- `backend/migrations/`
- `.env*` / any credential files
- `schemas/pg-phase0.sql`

If you must touch these, state it explicitly and expect a second reviewer. **`hotfix-approved` does not waive off-limits** — see [Rule precedence](#rule-precedence).

## Rule precedence

When rules appear to conflict, apply **the first matching row** (higher wins). Record the resolution in `openspec/changes/CYB-*/decisions.md` (or PR body for hotfix without a change dir).

| Priority | Rule | Examples |
|----------|------|----------|
| 1 | **User explicit instruction in the current message** | "只改 README" → docs-only path |
| 2 | **Safety & secrets** | Never commit credentials; never bypass auth on new paths |
| 3 | **Off-limits zones** | Hotfix touching `auth*` still needs explicit user approval + ≥2 reviewers |
| 4 | **Deploy-before-commit** | Runtime change → deploy dev + verify before commit (not waived by hotfix) |
| 5 | **OpenSpec / Linear traceability** | Runtime work needs CYB-xxx + change dir (except `hotfix-approved` skips **gate only**) |
| 6 | **CI hard gates** | commitlint, test-integration, openspec-gate (unless exempt label) |
| 7 | **Default workflow** | WORKFLOWS.md path for bug/feature/hotfix |

**Common conflicts:**

| Situation | Resolution |
|-----------|------------|
| Hotfix + must edit `auth*` | Proceed only after user confirms in chat; PR needs ≥2 reviewers; append `decisions.md` on backfill |
| User says "skip tests" | Run at least **Tier S** (lint/fmt); explain in `decisions.md` if skipping Tier M/L |
| `docs-only` but diff touches `backend/` | Not docs-only — full runtime path |
| Deploy gate vs "commit now" | Deploy verify first; commit only after user approves deploy |

## Verification tiers

Avoid running full `build` on every one-line fix. After each **accepted** code edit, run the **highest tier** that applies:

| Tier | When | Backend | Frontend | SDK |
|------|------|---------|----------|-----|
| **S — small** | ≤2 files, no router/handler/middleware/OpenAPI, no shared types | `make fmt && make vet` | `npm run lint` | `ruff check` on touched paths |
| **M — medium** | Default for most PRs; new/changed logic; >2 files or tests exist for package | Tier S + `go test` packages touched (`go test ./internal/foo/...`) | Tier S + `npm run test -- --run` (related tests if known) | Tier S + `pytest` for touched modules |
| **L — large** | Cross-module; UI routes; API contract; build/config; before PR / deploy | Tier M + `go test ./...` | Tier M + `npm run build` | Tier M + full `pytest tests/unit/` |

**Always Tier L before:** opening PR, deploy verification, or touching off-limits-adjacent code.

**Upgrade triggers (bump one tier):** changed `go.mod` / `package.json` deps; renamed exported symbols; modified `App.tsx` routes or shared `components/common/*`; any edit under `openspec/specs/` (baseline behavior — treat as **Tier L**).

## Decision log (audit trail)

Agents MUST append to `openspec/changes/CYB-{id}-{slug}/decisions.md` when:

- Resolving a [rule precedence](#rule-precedence) conflict
- Touching or requesting exception to off-limits
- Choosing verification Tier S when M would normally apply (user pressure or timebox)
- Skipping or deferring deploy verification (only with user-written approval in chat)
- **Chrome DevTools MCP unavailable** while diff touches `Frontend/` (document blocker; user may assist)

**Hotfix without change dir:** put the same entries in the PR description under `## Agent decisions`.

Format (append-only):

```markdown
## YYYY-MM-DD — Short title
- **Context**: …
- **Decision**: …
- **Alternatives**: …
- **Rationale**: …
```

## Rule maintenance

| What | Who | How |
|------|-----|-----|
| `docs/agents/**`, `openspec/config.yaml`, entry pointers | Any engineer; **review required** | PR titled `docs(agents): …` or bundled in feature PR when workflow changes; sync `config.yaml` `context`/`verification` with [`domain.md`](domain.md) when stack/modules change |
| `openspec/specs/**` (baseline behavior) | Feature owner + reviewer | Merged via Feature archive step |
| `.cursor/rules/*.mdc` | Maintainers only | Thin pointers to `docs/agents/` — no duplicate policy text |
| CI gates (`openspec-gate`, `commitlint`) | Platform / repo admins | Change only with team notice in PR |

**Principles:** one source of truth (`docs/agents/`); tool entry files stay pointers; user-facing README summarizes, links here for detail.

## Linear integration

- Every deliverable change MUST have a Linear Issue (create via MCP if missing)
- PR body MUST contain Linear ID (`CYB-xxx`) and OpenSpec change-id
- Update Linear state to Done after merge, include commit hash

## Skills and tools

- Trigger skills per scenario without being asked — see `docs/agents/SKILLS.md`
- MCP rules: `docs/agents/MCP-POLICY.md`

## Verification commands (reference)

Full commands by tier — see [Verification tiers](#verification-tiers).

| Module | Tier S | Tier M | Tier L |
|--------|--------|--------|--------|
| Backend | `make fmt && make vet` | + `go test ./path/to/pkg/...` | + `go test ./...` |
| Frontend | `npm run lint` | + `npm run test -- --run` | + `npm run build` |
| SDK | `uv run ruff check src/` | + `pytest` touched | + `pytest tests/unit/` |

## What NOT to do

- Do NOT ask users to run a ritual prompt like "prepare environment per project standards"
- Do NOT write runtime code before OpenSpec change exists (except `hotfix-approved` with documented backfill)
- Do NOT commit without deployment verification (deploy-before-commit rule)
- Do NOT ask the user to「浏览器点一遍」when the diff touches `Frontend/` and Chrome DevTools MCP is available — run MCP yourself on dev
- Do NOT introduce new dependencies without declaring them
- Do NOT bypass auth checks on new paths
- Do NOT refactor or clean up unrelated code
