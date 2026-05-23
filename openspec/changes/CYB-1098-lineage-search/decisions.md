# Decisions — CYB-1098

## 2026-05-23 — Complete existing endpoint skeleton
- **Context**: `backend/routes/routes.go` and `backend/internal/handlers/audit/handler.go` already contain an early `/api/v1/audit/lineage-search` route and handler, but CYB-1098 remains incomplete and the endpoint is absent from OpenAPI/api-guide/smoke coverage.
- **Decision**: Treat CYB-1098 as completing and hardening the existing skeleton rather than adding a duplicate endpoint.
- **Alternatives**: Delete the skeleton and reimplement from scratch through a new repository abstraction.
- **Rationale**: Reusing the existing route keeps the change scoped while still requiring contract sync, tests, and deploy verification.

## 2026-05-23 — Default to dependency relation types
- **Context**: `asset_relations` stores several edge families, including dependency/structural lineage and versioning edges such as `revision_of`. The historical compliance-lineage design calls out dependency relations such as `split_from`, `contains`, and `derived_from`.
- **Decision**: Traverse dependency/structural relation types by default, return relation metadata, and support an explicit `relation_types` filter for narrower searches.
- **Alternatives**: Traverse every `asset_relations` row by default, or restrict lineage search to one relation type.
- **Rationale**: Compliance lineage should include asset dependency flow without silently mixing unrelated edge semantics into default results.

## 2026-05-23 — SDK and Frontend out of scope
- **Context**: CYB-1098 asks for an API endpoint only; no current frontend screen or SDK parity requirement is attached to the issue.
- **Decision**: Do not modify `Frontend/` or `sdk/` in this issue unless later user instruction changes scope.
- **Alternatives**: Add a Python SDK client and UI surface immediately.
- **Rationale**: Adding SDK or UI would expand verification/deploy scope beyond this backend API checkpoint.

## 2026-05-23 — OpenSpec checkpoint approved
- **Context**: OpenSpec checkpoint was presented after CYB-1098 proposal, design, tasks, decisions, context files, and spec delta were created.
- **Decision**: User approved continuing with runtime implementation in chat: "可以的".
- **Alternatives**: Keep waiting at the checkpoint.
- **Rationale**: Repository workflow requires explicit approval before editing runtime paths.

## 2026-05-23 — Complex dev fixture verification
- **Context**: User asked to mock more complex data and verify business logic through API. Public API can create MCAP files and assets, but there is no public endpoint to write arbitrary `asset_relations` dependency edges.
- **Decision**: Create fixture assets through the deployed API, then insert isolated `asset_relations` rows directly in dev PostgreSQL with `cyb1098-smoke` metadata and verify behavior through `GET /api/v1/audit/lineage-search`.
- **Alternatives**: Only use existing dev data, or add a temporary relation-write endpoint.
- **Rationale**: Existing data did not guarantee multi-parent/cycle/relation-filter coverage, and adding a temporary public write API would expand runtime scope incorrectly.
