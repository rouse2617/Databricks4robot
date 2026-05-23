# Agent Runtime Guardrails

Portable runtime contract for all AI agents and automation in this repository (Cursor, Codex, Claude Code, CI, Git hooks).

**Policy** lives in [`AI-RULES.md`](AI-RULES.md). **This file** defines how agents and scripts consume that policy at runtime — without duplicating workflow text.

---

## Principles

1. **Single source of truth** — Change behavior in `docs/agents/` only. Tool entry files (`AGENTS.md`, `CLAUDE.md`, `.cursor/rules/*.mdc`) are pointers.
2. **Tool-neutral core** — Shared scripts live under `scripts/agent-harness/`. Optional client hooks call these scripts; they do not own policy.
3. **Hard vs soft enforcement** — Model memory is unreliable. Git hooks and CI are the hard backstop; agent-time scripts provide early feedback.
4. **Compact recovery** — After context compaction, re-read [`AI-RULES.md`](AI-RULES.md) and [`.agent/context/current-work.md`](../../.agent/context/current-work.md).

---

## Layout

```text
docs/agents/
├── AI-RULES.md          # Policy (SSOT)
├── HARNESS.md           # This file — runtime phases & adapter rules
└── TOOL-ENTRY.md        # Per-IDE entry map

.agent/context/
└── current-work.md      # Active CYB / branch / constraints (human-updated)

scripts/agent-harness/
├── after-edit.sh        # Advisory checks after file edits
└── before-commit.sh     # Hard checks before git commit
```

---

## Runtime phases

| Phase | Script (when wired) | Purpose | Default severity |
|-------|---------------------|---------|------------------|
| `session-start` | _(optional adapter)_ | Load `AI-RULES.md`, `SETUP.md`, `current-work.md` | Informational |
| `after-edit` | `scripts/agent-harness/after-edit.sh` | Path-based reminders (API sync, migrations, frontend verify) | Advisory (`exit 0`) |
| `before-command` | _(reserved)_ | Dangerous shell command interception | Adapter-defined |
| `before-commit` | `scripts/agent-harness/before-commit.sh` | Block credentials, minimum API contract artifacts | Hard (`exit 1`) |
| `stop-check` | _(reserved)_ | Task completeness before claiming "done" | Adapter-defined |

### Failure semantics

- **Advisory** — Print grouped findings to stderr; exit `0` unless a clearly dangerous pattern is detected (e.g. staged `.env`).
- **Hard** — Print actionable error; exit non-zero. Used by `before-commit.sh` and Git/CI wiring.

Do **not** assume every agent runtime supports the same hook system. When hooks are unavailable, agents should run the scripts manually at the matching phase.

---

## Adapter rules

| Consumer | Role |
|----------|------|
| **Cursor** | `.cursor/rules/*.mdc` → link to `AI-RULES.md` / `HARNESS.md` only |
| **Codex** | `AGENTS.md` → pointer |
| **Claude Code** | `CLAUDE.md` → pointer; optional `.claude/settings.json` may invoke shared scripts |
| **Git** | `.githooks/commit-msg` (Conventional Commits) + optional `pre-commit` calling `before-commit.sh` |
| **CI** | May call `before-commit.sh` on changed paths in PR checks |

**Forbidden:** Copying policy paragraphs into adapter files. **Forbidden:** Making `.claude/` paths required for core workflow.

---

## When to run

| You changed | Run after edit | Run before commit |
|-------------|----------------|-------------------|
| `backend/routes/`, handlers | Yes | Yes (API contract minimum) |
| `api/openapi.yaml` | Yes | Yes if handlers also changed |
| `backend/migrations/` | Yes (warn) | Yes (warn) |
| `Frontend/` | Yes | Yes (reminder; deploy gate in `deploy-before-commit.md`) |
| `sdk/` | Yes | Optional |
| `docs/agents/` only | Optional | Optional |

Full deploy and verification requirements remain in [`deploy-before-commit.md`](deploy-before-commit.md) and [`deploy-verification.md`](deploy-verification.md).

---

## Related

- [`AI-RULES.md`](AI-RULES.md) — automatic behavior, API contract sync, verification tiers
- [`TOOL-ENTRY.md`](TOOL-ENTRY.md) — read order per IDE
- [`.agent/context/current-work.md`](../../.agent/context/current-work.md) — active iteration scratchpad
- Linear **CYB-1115** — Agent Runtime Guardrails (milestone: AI Coding Governance)
