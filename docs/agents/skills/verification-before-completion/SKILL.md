---
name: verification-before-completion
description: Use before claiming work is complete, tests pass, deploy OK, or asking the user to commit — run verification commands and cite evidence. Maps to deploy-verification §6 and AI-RULES verification tiers.
---

# Verification Before Completion

**Core principle:** Evidence before claims. No exceptions before commit, PR, or「可以 commit 吗？」.

Adapted from [obra/superpowers](https://github.com/obra/superpowers) `verification-before-completion`, wired to this repo's deploy gate.

---

## Gate (run every time)

```
1. IDENTIFY  — What command proves this claim?
2. RUN       — Full command, fresh (this session)
3. READ      — Exit code + full relevant output
4. VERIFY    — Output matches claim?
5. CLAIM     — Only then state pass/fail, with evidence snippet
```

Skip a step = invalid completion.

---

## cyber-databrew command map

| Claim | Required evidence | Not sufficient |
|-------|-------------------|----------------|
| Lint clean | `make fmt && make vet` / `npm run lint` / `ruff check` — 0 errors | "looks fine" |
| Unit tests pass | `go test …` / `npm run test` / `pytest` — 0 failures | Earlier run |
| Build OK | `npm run build` (Tier L) / `go build` if you changed build tags | Linter only |
| Bug fixed | Reproduce original symptom on **dev** (or minimal repro) — passes | Code edited |
| Deploy OK | Image tag `:<git-sha>` + Cloud Run revision + URL in `tasks.md` or PR | Only `:cloudrun-dev-latest` or local build without deploy |
| Frontend regression | **§6.1** pages per [`deploy-verification.md`](../../deploy-verification.md#61-前端固定页面回归全站基线) — Chrome DevTools MCP when diff touches `Frontend/` | "should work on dev" |
| Backend regression | **§6.2** L1 or L2 per risk — curl output or `api-guide-smoke.sh` summary | Single `healthz` only when L2 required |
| OpenSpec tasks done | Re-read `tasks.md` checkboxes vs actual work | Tests pass |

Full tier table: [`AI-RULES.md`](../../AI-RULES.md#verification-tiers).

Fixed regression lists: [`deploy-verification.md`](../../deploy-verification.md#六固定回归清单团队基线) **§6.1–6.3**.

---

## Deploy verification checklist (§6)

After `deploy/cloudrun/*-dev.sh`, before asking user to commit:

### Frontend (`Frontend/` in diff)

- [ ] Record `FRONTEND_DEV_URL` + image tag (`:<sha>`) + revision (see `deploy-before-commit.md`)
- [ ] Chrome DevTools MCP: paths from `tasks.md` / `context-files.md` proposal scope
- [ ] §6.1: run the **minimum page set** for touched dirs (not necessarily 11/11)
- [ ] Screenshot → `openspec/changes/CYB-*/deploy-verify-*.png` when UI-visible change
- [ ] Console: no new `error` / unhandled rejection

### Backend (`backend/` in diff, no Frontend)

- [ ] Record `BASE` + image tag (`:<sha>`) + revision
- [ ] L1: `healthz` + core APIs (see §6.2 L1 block in deploy-verification)
- [ ] L2: `bash scripts/api-guide-smoke.sh` when handler/router/middleware/OpenAPI touched
- [ ] Paste smoke summary or exit code in PR / `tasks.md`

### Full stack

- [ ] Backend L1/L2 first, then frontend deploy, then §6.3 combined checklist

---

## Red flags — STOP

- "Should", "probably", "seems to", "Great!", "Done!" before running commands
- Commit / push / PR without deploy evidence (runtime paths)
- Trusting subagent "success" without VCS diff + verification
- Skipping §6 because "small UI tweak"
- Asking user to click through UI when Chrome DevTools MCP is available and diff touches `Frontend/`

---

## When blocked

Log in `openspec/changes/CYB-*/decisions.md` (or PR **Agent decisions** for hotfix):

- **Context**: what you tried
- **Decision**: blocked on X
- **Rationale**: MCP unavailable / env / IAP / etc.

Do **not** claim verification complete.

---

## Related

- Bugs during verification: [`systematic-debugging`](../systematic-debugging/SKILL.md)
- Hard bugs / repro loops: `.agents/skills/diagnose/SKILL.md` (if installed)
- Authoring tasks/deploy section: [`spec-writing-skill.md`](../../spec-writing-skill.md)
