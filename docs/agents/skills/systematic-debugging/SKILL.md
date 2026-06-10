---
name: systematic-debugging
description: Use when encountering any bug, test failure, or unexpected behavior before proposing fixes — root cause first, then minimal fix. Complements the diagnose skill for cyber-databrew.
---

# Systematic Debugging

**Core principle:** No fixes without root cause investigation first. Symptom patches create rework.

Adapted from [obra/superpowers](https://github.com/obra/superpowers) `systematic-debugging`.

**With this repo:** Use **this skill** for the four-phase discipline. Use **`.agents/skills/diagnose`** when you need a concrete feedback loop (failing test, curl script, MCP repro) — see [diagnose pairing](#pairing-with-diagnose) below.

---

## Iron law

```
NO FIXES WITHOUT ROOT CAUSE INVESTIGATION FIRST
```

Complete Phase 1 before editing production code.

---

## Four phases

### Phase 1 — Root cause

1. Read errors completely (stack trace, line numbers, API body).
2. Reproduce on **dev** (or minimal local repro) — note exact steps.
3. `git diff` / recent commits / dependency changes.
4. **Multi-layer systems** (Frontend → API → PG → ES): instrument each boundary once; see where data breaks before guessing.
5. Trace bad values **upstream** (who passed wrong filter/state?).

### Phase 2 — Pattern

1. Find **working** similar code in-repo (same hook, handler, reducer pattern).
2. Read reference implementation fully — don't skim.
3. List **every** difference between working and broken.
4. Check config: `tag_registry.yaml`, `algo_registry.yaml`, env, feature flags.

### Phase 3 — Hypothesis

1. One hypothesis: "X is root cause because Y".
2. **Smallest** test change to confirm/deny.
3. Failed? New hypothesis — don't stack fixes.
4. After **3 failed fixes**: stop — likely architecture issue; discuss with user (log in `decisions.md`).

### Phase 4 — Implementation

1. Failing test or repro script **first** (see `tdd` skill if behavior change).
2. One fix for root cause — no drive-by refactors.
3. Run verification tier per [`AI-RULES.md`](../../AI-RULES.md#verification-tiers).
4. Before commit: [`verification-before-completion`](../verification-before-completion/SKILL.md) + deploy §6 if runtime.

---

## cyber-databrew hotspots

| Symptom area | First checks |
|--------------|--------------|
| Assets list wrong | `useAssetsDiscoveryReducer`, backend `queries/run`, ES vs PG |
| Auth 401/403 | `X-Databrew-Token`, middleware — **off-limits** without approval |
| Search stale | outbox / ES sync — **off-limits** without approval |
| UI state | React reducer + API response shape mismatch (client re-filter?) |

Read `context-files.md` in the active OpenSpec change for paths the planner already curated.

---

## Red flags — return to Phase 1

- "Quick fix for now"
- "Just try X"
- Multiple edits before one verification run
- "Probably the cache"
- Fix #4 after three failures

---

## Pairing with diagnose

| Situation | Skill |
|-----------|--------|
| Need disciplined **process** (phases, when to stop) | **systematic-debugging** (this file) |
| Need a **fast pass/fail signal** (test, curl, Playwright) | **diagnose** |
| Ready to claim fix + deploy | **verification-before-completion** |

---

## Related

- [`deploy-verification.md`](../../deploy-verification.md) — regression while verifying fixes on dev
- [`spec-writing-skill.md`](../../spec-writing-skill.md) — capture root cause in `decisions.md` if you skipped OpenSpec first (hotfix backfill)
