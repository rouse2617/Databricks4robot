# Decisions

## 2026-07-23 — Defer Linear traceability at user request

- **Context**: Repository workflow requires a CYB issue, but the user explicitly asked to skip Linear and backfill it later.
- **Decision**: Use temporary `CYB-TBD` naming for the isolated OpenSpec branch/change and make replacing it a blocking pre-commit task. Do not create or update a Linear issue.
- **Alternatives**: Stop all design work until a CYB issue exists.
- **Rationale**: The current explicit user instruction has higher precedence, while preventing a placeholder from reaching commit or PR preserves traceability requirements.

## 2026-07-23 — Propose but do not implement an off-limits migration

- **Context**: Target-authoritative idempotency and audit require durable prod state, and `backend/migrations/` is off-limits without explicit approval.
- **Decision**: Specify a `pipeline_promotions` Atlas migration, but stop for explicit user approval before creating or changing any migration file; require a second reviewer during implementation.
- **Alternatives**: Store audit only in dev or in process memory.
- **Rationale**: Neither alternative can guarantee target-side idempotency, atomic retry behavior, or durable production audit.

## 2026-07-23 — OpenSpec and migration implementation approved

- **Context**: The OpenSpec checkpoint required approval before runtime edits, and the proposed `pipeline_promotions` migration is in an off-limits directory that requires explicit user approval.
- **Decision**: Treat the user's “ok，开发吧” response to the approval request as approval to implement the reviewed OpenSpec, including the explicitly named `pipeline_promotions` Atlas migration. A second reviewer remains required before merge.
- **Alternatives**: Implement only non-schema work and pause again before the migration.
- **Rationale**: The approval directly followed a request that named both the OpenSpec and migration approval; proceeding matches the reviewed scope without expanding it.
