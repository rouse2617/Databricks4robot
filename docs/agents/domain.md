# Domain Docs

How engineering skills should consume domain documentation in **cyber-databrew**.

## Sync with `openspec/config.yaml`

`openspec/config.yaml` → `context:` injects stack, module map, and off-limits into OpenSpec planning. **When modules, tech stack, or off-limits change**, update **both** this file (glossary / pointers) and `config.yaml` `context` + `verification` so agents do not see conflicting facts.

Keep `context` as a **summary**; deep API/field detail stays in `docs/review/*`, not duplicated in config.

## Before exploring, read these

- **`CONTEXT.md`** at the repo root (create via `/grill-with-docs` when terms crystallise; file may not exist yet).
- **`docs/review/*`** — integration guides and schema companions (`sql.md`, `api-guide.md`, data model docs). These are detailed references; prefer `CONTEXT.md` terms in short agent output.

If `CONTEXT.md` is missing, proceed without blocking; `/grill-with-docs` adds terms lazily.

## Use the glossary’s vocabulary

When naming domain concepts (issues, refactors, tests), prefer terms from `CONTEXT.md` once it exists (e.g. **asset**, **MCAP**, **delivery**, **outbox relay**, **lakehouse** vs ad-hoc synonyms).

## Flag conflicts

If a proposal contradicts established domain decisions, call it out explicitly rather than silently overriding.
