# Domain Docs

How engineering skills should consume domain documentation in **cyber-databrew**.

## Before exploring, read these

- **`CONTEXT.md`** at the repo root (create via `/grill-with-docs` when terms crystallise; file may not exist yet).
- **`docs/adr/`** — e.g. `001-iceberg-catalog-selection.md` for lakehouse/catalog decisions.
- **`docs/review/*`** — integration guides and schema companions (`sql.md`, `api-guide.md`, data model docs). These are detailed references; prefer `CONTEXT.md` terms in short agent output.

Layout: **single-context** (one glossary + repo-wide ADRs).

If `CONTEXT.md` is missing, proceed without blocking; `/grill-with-docs` adds terms lazily.

## Use the glossary’s vocabulary

When naming domain concepts (issues, refactors, tests), prefer terms from `CONTEXT.md` once it exists (e.g. **asset**, **MCAP**, **delivery**, **outbox relay**, **lakehouse** vs ad-hoc synonyms).

## Flag ADR conflicts

If a proposal contradicts an ADR under `docs/adr/`, call it out explicitly rather than silently overriding.
