# Available Skills

Skills are reusable expert workflows. Read and follow the SKILL.md file when a matching scenario is triggered.

**In-repo doc skills** (under `docs/agents/`, not `.agents/skills/`) are first-class — same must-use rules as packaged skills.

## Must-use (trigger automatically when scenario matches)

| Scenario | Skill | Path |
|----------|-------|------|
| Creating or modifying OpenSpec artifacts | spec-writing (in-repo doc) | [`spec-writing-skill.md`](spec-writing-skill.md) |
| Before commit / deploy OK /「完成」claims | verification-before-completion — run commands, cite SHA tag + revision evidence | [`skills/verification-before-completion/SKILL.md`](skills/verification-before-completion/SKILL.md) → deploy-verification **§6** |
| Bug, test failure, unexpected behavior (before fixing) | systematic-debugging — root cause phases before patching; pairs with `diagnose` for repro | [`skills/systematic-debugging/SKILL.md`](skills/systematic-debugging/SKILL.md) |
| Debugging a bug or failure | diagnose | `.agents/skills/diagnose/SKILL.md` |
| Behavior change that needs tests | tdd | `.agents/skills/tdd/SKILL.md` |
| Updating Linear progress | linear-progress-update | `.claude/skills/linear-progress-update/SKILL.md` |
| Breaking a plan into Linear issues | to-issues | `.agents/skills/to-issues/SKILL.md` |

## Recommended (use when beneficial)

| Scenario | Skill | Path |
|----------|-------|------|
| Stress-testing a plan or design | grill-me | `.agents/skills/grill-me/SKILL.md` |
| Challenging plan against domain docs | grill-with-docs | `.agents/skills/grill-with-docs/SKILL.md` |
| Improving architecture | improve-codebase-architecture | `.agents/skills/improve-codebase-architecture/SKILL.md` |
| Handing off context to another agent | handoff | `.agents/skills/handoff/SKILL.md` |
| Converting context into a PRD | to-prd | `.agents/skills/to-prd/SKILL.md` |
| Triaging incoming issues | triage | `.agents/skills/triage/SKILL.md` |
| Generating user-facing changelogs from commits | changelog-generator | [`skills/changelog-generator/SKILL.md`](skills/changelog-generator/SKILL.md) |
| 通过 Pub/Sub 下发资产到 Databrew(触发订阅任务)| dispatch-asset — 消息格式、单/批语义、gcloud/Python 示例、验证步骤 | [`skills/dispatch-asset/SKILL.md`](skills/dispatch-asset/SKILL.md) |

## Restricted (conditional use only)

| Skill | Condition |
|-------|-----------|
| prototype | Output is throwaway only; NEVER merge prototype code to main |
| write-a-skill | Only when explicitly asked to create a new skill |

## Installing skills

If `.agents/skills/` is missing or incomplete:

```
npx skills@latest add mattpocock/skills --agent <cursor|codex|...> -y --copy
```

**Lock file (local only, not in git):**

| Item | Value |
|------|--------|
| Path | Repository root: `./skills-lock.json` |
| Git | Listed in `.gitignore` — each developer/agent machine has its own copy |
| Created by | The same `npx skills@latest add …` command above |
| Verify | `test -f skills-lock.json && grep -E '"(tdd|diagnose|grill-me)"' skills-lock.json` |
| Fallback | If lock missing, check directories exist: `ls .agents/skills/tdd .agents/skills/diagnose` |

Do not commit `skills-lock.json`; CI does not depend on it.
