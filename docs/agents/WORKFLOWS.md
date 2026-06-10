# Standard Workflows

**Agents:** Infer the path from the user's message and execute all steps automatically. Users do not name these paths.

**Authority:** Step tables below are **workflow order** only. Automatic behavior, rule conflicts, and precedence → [`AI-RULES.md`](AI-RULES.md)（especially [Rule precedence](AI-RULES.md#rule-precedence)）. If this file and AI-RULES conflict, **AI-RULES wins**.

| Signal in user message | Path |
|------------------------|------|
| Bug, fix, broken, regression, 不工作, 报错 | Bug Fix |
| Feature, 新功能, add API, 新增 | Feature |
| P0, hotfix, 线上紧急 | Hotfix |
| docs only, README, 只改文档 | Skip OpenSpec for runtime; label `docs-only` |
| CI/Tekton only | label `infra-ci-only` |

Three paths depending on change type. Follow the matching path strictly — **without asking permission**.

## Bug Fix Path

| Step | Action | Hard requirement |
|------|--------|-----------------|
| 1 | Create Linear Issue (label: `bug`) | Title + reproduction steps |
| 2 | Create OpenSpec change | `proposal.md` + `tasks.md` + at least 1 spec delta |
| 3 | **OpenSpec checkpoint** | User confirms OpenSpec OK → then branch + code ([`AI-RULES.md`](AI-RULES.md) step 4) |
| 4 | Branch: `fix/CYB-{id}-*` | Name must contain Linear ID |
| 5 | Code + local verify | Tier per AI-RULES; **new/changed HTTP API** → [API contract sync](AI-RULES.md#api-contract-sync-mandatory) same PR |
| 6 | Push + create PR (fill template completely) | Linear ID + OpenSpec change-id + test evidence |
| 7 | CI passes | openspec-gate + commitlint + test-integration + pre-commit |
| 8 | Deploy verification | `apply-migration-dev.sh` (if migration) → build → push → dev; smoke via [`deploy-verification.md`](deploy-verification.md) §2.0 |
| 9 | Review (>=1 person, within 24h) | Approve then merge |
| 10 | Squash merge | `fix(scope): description` |
| 11 | Linear → Done | Include commit hash |

**Exemption**: Bug path may skip `design.md`.

## Feature Path

| Step | Action | Hard requirement |
|------|--------|-----------------|
| 1 | Create Linear Issue (label: `feature`) | Goal + implementation approach |
| 2 | Create OpenSpec change (full) | `proposal.md` + `design.md` + `tasks.md` + spec delta |
| 3 | **OpenSpec checkpoint** | User confirms OpenSpec OK (required even if `spec-approved` label exists) |
| 4 | **Spec Review** (optional, before or after checkpoint) | Label `needs-spec-review` → reviewer confirms → `spec-approved` |
| 5 | Branch: `feat/CYB-{id}-*` | — |
| 6 | Code (split into sub-PRs if > 200 lines) | Each step: verification tier per AI-RULES; API changes include contract files |
| 7 | Push + create PR | Same as above + spec delta scope description |
| 8 | CI passes | Same + openapi-diff (if API changed) |
| 9 | Deploy verification | Same as Bug §2.0 scripts |
| 10 | Review (complex features: >=2 people) | — |
| 11 | Squash merge | `feat(scope): description` |
| 12 | Archive in repo | Merge spec delta into `openspec/specs/`, remove `changes/` dir (**mandatory**, no CLI) |
| 13 | Linear → Done | Summary + commit hash |

**Risk gate**: touching off-limits zones → automatically requires second reviewer.

## Hotfix Emergency Path

| Step | Action |
|------|--------|
| 1 | Create Linear Issue (label: `hotfix-approved`) |
| 2 | **Skip** OpenSpec change (CI gate recognizes label) |
| 3 | Branch: `hotfix/CYB-{id}-*` |
| 4 | Code + local verify | Tier per AI-RULES; PR `## Agent decisions` if off-limits or precedence conflict |
| 5 | Push + PR (mark "hotfix, spec to follow") |
| 6 | CI passes (openspec-gate skipped) |
| 7 | Deploy verification | 含 `Frontend/` 则 MCP；否则 smoke（同 Bug） |
| 8 | Quick review (>=1 person) |
| 9 | Squash merge |
| 10 | **Within T+2 business days**: create OpenSpec change retroactively (auto-reminder, alert on overdue) |

## Label-based exemptions (CI skips openspec-gate)

| Label | Scope | Condition |
|-------|-------|-----------|
| `docs-only` | Pure documentation changes | No runtime code touched |
| `infra-ci-only` | CI/Tekton/deploy scripts | No runtime behavior change |
| `hotfix-approved` | Production emergency | Must backfill spec within T+2 |
