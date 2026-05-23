# Decisions — CYB-1100

## 2026-05-23 — OpenSpec checkpoint only
- **Context**: User asked to create the OpenSpec checkpoint only for CYB-1100 and explicitly forbade runtime edits under `backend/`, `Frontend/`, `sdk/`, and `dagster/`.
- **Decision**: Limit this turn to OpenSpec artifacts and `.agent/context/current-work.md`.
- **Alternatives**: Implement the existing handler skeleton immediately.
- **Rationale**: This follows the repository OpenSpec gate and the user's explicit scope.

## 2026-05-23 — Complete existing route skeleton
- **Context**: `backend/routes/routes.go` already registers `GET /api/v1/logical-assets/:id/ratings-history`, and `backend/internal/handlers/asset/handler.go` has an early `HandleRatingsHistory` implementation.
- **Decision**: Treat CYB-1100 as contracting, testing, documenting, and hardening the existing endpoint rather than adding a second route.
- **Alternatives**: Add a new handler or a differently named endpoint.
- **Rationale**: Reusing the existing route keeps behavior aligned with Linear and avoids duplicate public API surface.

## 2026-05-23 — Ratings source for CYB-1100
- **Context**: Existing code exposes rating-like algorithm fields from `asset_algo_latest`, but product docs define quality ratings as `asset_metrics` rows with `metric_key` in the `rating.*` namespace and call out this endpoint for cross-version comparisons.
- **Decision**: Define CYB-1100 ratings history as cross-revision `asset_metrics` `rating.*` rows, grouped per revision. `asset_algo_latest` remains algorithm state and is not part of this checkpoint contract.
- **Alternatives**: Reuse the current `asset_algo_latest` skeleton, or merge `asset_algo_latest`, `asset_metrics`, `asset_eval_results`, and `asset_tags.quality` in one response.
- **Rationale**: The endpoint name and data-model docs point to quality/rating trend data, not algorithm current-state projection. Fixing the skeleton to the documented source avoids shipping a misleading API.

## 2026-05-23 — No ratings is not not-found
- **Context**: A promoted revision can exist before algorithm ratings are written.
- **Decision**: Existing logical asset families return `200` with one item per revision and empty `ratings` arrays when no rating rows exist; missing logical assets return `404 ASSET_NOT_FOUND`.
- **Alternatives**: Return `200 items: []` for both cases, or omit unrated revisions.
- **Rationale**: Clients need to distinguish a valid but unrated revision from an invalid logical asset id.

## 2026-05-23 — OpenSpec checkpoint approved
- **Context**: CYB-1100 OpenSpec checkpoint was prepared before runtime edits and the worktree was rebased onto latest `origin/dev`.
- **Decision**: User approved continuing with runtime implementation in chat: "可以的".
- **Alternatives**: Keep waiting at the checkpoint.
- **Rationale**: Repository workflow requires explicit approval before editing runtime paths.

## Open Questions
- Should a future issue add algorithm state (`asset_algo_latest.result_score/result_tag`) as a separate source type, or keep this endpoint strictly to `rating.*` metrics?
- Should SDK parity be required for logical asset endpoints as a separate issue, even though CYB-1100 is scoped as API-only?
