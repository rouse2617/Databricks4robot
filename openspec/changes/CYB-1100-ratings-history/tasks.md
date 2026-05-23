# Tasks — CYB-1100

## Context files
- `backend/routes/routes.go` — existing logical asset route registration.
- `backend/internal/handlers/asset/handler.go` — current ratings-history skeleton and asset handler error patterns.
- `backend/internal/usecase/asset/versioning.go` — logical asset current revision semantics and not-found errors.
- `backend/internal/usecase/asset/provenance.go` — revision list ordering and logical family conventions.
- `backend/internal/postgres/repos.go` — `ListByLogicalAssetID` and `asset_tags` read patterns.
- `backend/internal/postgres/eval_repos.go` — `asset_metrics` row model and metric read patterns.
- `backend/internal/models/schema_evolution.go` — `AssetAlgoLatest` and `AssetTag` JSON fields.
- `docs/review/unified-asset-catalog/archive/prd-rev11-historic.md` — `rating.*` semantics and ratings-history intent.
- `docs/review/eval-metrics-design.md` — `asset_metrics` schema and metric source fields.
- `api/openapi.yaml` — public API contract.
- `docs/review/api-guide.md` — curl documentation.
- `scripts/api-guide-smoke.sh` — dev smoke coverage.

## Implementation
- [x] [backend] Validate `logical_asset_id` path param: trim whitespace; reject blank/malformed values with `400 INVALID_ARGUMENT`.
- [x] [backend] Return `404 ASSET_NOT_FOUND` when the logical asset has no non-deleted revision rows.
- [x] [backend] Return grouped response shape: `logical_asset_id`, `items`, `count`, with one item per revision.
- [x] [backend] Include revision metadata: `asset_id`, `revision`, `is_current`, `lifecycle_state`, `created_at`.
- [x] [backend] Attach rating rows under `items[].ratings[]` from `asset_metrics` where `metric_key LIKE 'rating.%'`, including metric key/value fields, source fields, eval identity, target identity, run id, confidence, and recorded time.
- [x] [backend] Preserve revisions with no ratings as `ratings: []`.
- [x] [backend] Ensure deterministic ordering: revisions by `revision ASC, created_at ASC`, ratings by `metric_key ASC, source_type ASC, source_name ASC, recorded_at ASC`.
- [x] [backend] Keep implementation read-only; do not touch migrations, auth middleware, outbox, `Frontend/`, `sdk/`, or `dagster/`.

## Tests
- [x] [backend] Invalid or blank logical id returns `400 INVALID_ARGUMENT`.
- [x] [backend] Missing logical asset returns `404 ASSET_NOT_FOUND`.
- [x] [backend] Existing logical asset with no rating rows returns `200`, revision items, and empty `ratings` arrays.
- [x] [backend] Multiple revisions with rating rows are grouped by revision and sorted oldest-to-newest.
- [x] [backend] Multiple rating metric rows on one revision are sorted deterministically.
- [x] [backend] Deleted revision rows are excluded.

## API contract sync
See [`docs/agents/AI-RULES.md` § API contract sync](../../docs/agents/AI-RULES.md#api-contract-sync-mandatory).

- [x] `api/openapi.yaml` — add `/api/v1/logical-assets/{id}/ratings-history` path, query/path params, schemas, and errors.
- [x] `docs/review/api-guide.md` — add curl example, success response, no-ratings example, and 400/404 notes.
- [x] `scripts/api-guide-smoke.sh` or `scripts/smoke-ratings-history-dev.sh` — add success and error-path checks.
- [x] `openspec/changes/CYB-1100-ratings-history/specs/asset-management/spec.md` — behavior delta.
- [ ] SDK — out of scope unless user explicitly asks for SDK parity in this issue.
- [ ] Frontend — out of scope; no UI consumes the endpoint in this issue.

## Local verification (Tier L per AI-RULES because this adds a public HTTP API contract)
- [x] `gofmt` touched Go files.
- [x] `cd backend && go vet ./... && go test ./...`.
- [x] `bash -n scripts/api-guide-smoke.sh` or `bash -n scripts/smoke-ratings-history-dev.sh`.
- [x] `git diff --check`.
- [x] `scripts/agent-harness/after-edit.sh`.

## Deploy verification (before commit — runtime only)
- [x] Build backend image with git SHA tag and `cloudrun-dev-latest`.
- [x] Push backend image tags.
- [x] Deploy backend dev using SHA-tagged image and record image tag, Cloud Run revision, and URL.
- [x] Source `scripts/dev-backend-env.sh` and run targeted smoke for:
  - successful ratings history query for a known logical asset
  - missing logical asset or invalid id error path
  - existing logical asset revision with no ratings, if dev data has one
- [x] Run relevant API guide smoke checks on dev.

Dev fixture inserted into Cloud SQL for verification:
- `CYB1100A`: 3 revisions (`CYB10A01`, `CYB10A02`, `CYB10A03`) with mixed `rating.*` metrics and one non-rating `blur_ratio` metric to verify exclusion.
- `CYB1100B`: 1 revision (`CYB10B01`) with no rating metrics to verify `ratings: []`.

Verification evidence:
- Targeted Cloud Run API assertions passed for `CYB1100A` multi-revision grouping, sorted `rating.*` rows, `blur_ratio` exclusion, `CYB1100B` empty ratings, invalid id `400 INVALID_ARGUMENT`, and missing id `404 ASSET_NOT_FOUND`.
- `scripts/api-guide-smoke.sh` passed the CYB-1100 ratings-history success check and invalid id check. Full script ended with the known Cloud Run root `/healthz` 404; endpoint checks passed.

### Deploy record — CYB-1100
| Service | Image tag | Cloud Run revision | URL |
|---------|-----------|-------------------|-----|
| backend-dev | `cyber-databrew-backend:519257b-cyb1100` | `cyber-databrew-backend-dev-00168-fxp` | `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app` |

## PR
- [ ] PR template filled with Linear `CYB-1100`, OpenSpec change id, tests, and deploy evidence.
- [ ] Linear updated with commit/deploy summary after merge.
