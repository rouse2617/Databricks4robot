# Tasks — CYB-1098

## Context files
- `backend/internal/handlers/audit/handler.go` — existing audit lineage handler skeleton.
- `backend/routes/routes.go` — current `/api/v1/audit/lineage-search` route registration.
- `schemas/pg-phase0.sql` — `asset_relations` columns and indexes.
- `api/openapi.yaml` — public API contract target.
- `docs/review/api-guide.md` — curl documentation target.
- `scripts/api-guide-smoke.sh` — dev contract smoke target.

## OpenSpec checkpoint
- [x] [openspec] Create CYB-1098 proposal, design, tasks, context files, decisions, and spec delta before runtime edits.
- [x] [Linear] Inspect CYB-1098 and move it to In Progress.

## Implementation
- [x] [backend] Add focused tests for lineage query validation (`asset_id`, `direction`, `depth`).
- [x] [backend] Add focused tests for upstream traversal from child asset to parent ancestors.
- [x] [backend] Add focused tests for downstream traversal from parent asset to child descendants.
- [x] [backend] Add focused tests for `both` traversal response shape and deterministic ordering.
- [x] [backend] Add focused tests for depth limiting and cycle-safe traversal.
- [x] [backend] Add focused tests for empty results returning `200`, empty `nodes`, and `count: 0`.
- [x] [backend] Harden `/api/v1/audit/lineage-search` handler behavior against nil storage, query errors, scan errors, and row iteration errors.
- [x] [backend] Ensure recursive CTE traversal uses `asset_relations` dependency relation types by default and returns relation metadata.
- [x] [backend] Support a validated optional `relation_types` filter for narrower lineage searches.
- [x] [backend] Keep implementation read-only; do not touch migrations, auth middleware, outbox internals, or unrelated runtime paths.

## API contract sync
- [x] `api/openapi.yaml` — add `/api/v1/audit/lineage-search` path, query params including `relation_types`, response schema, and errors.
- [x] `docs/review/api-guide.md` — add lineage search curl examples, response examples, empty result notes, relation type semantics, and validation errors.
- [x] `scripts/api-guide-smoke.sh` — add success and validation-error checks.
- [x] `openspec/changes/CYB-1098-lineage-search/specs/search/spec.md` — behavior delta.
- [ ] SDK — out of scope unless user explicitly asks for SDK parity in this issue.
- [ ] Frontend — out of scope; no UI consumes the endpoint in this issue.

## Local verification (Tier L per AI-RULES because this adds public HTTP API contract)
- [x] `gofmt` touched Go files.
- [x] `cd backend && go vet ./... && go test ./...`.
- [x] `bash -n scripts/api-guide-smoke.sh`.
- [x] `git diff --check`.
- [x] `scripts/agent-harness/after-edit.sh`.

## Deploy verification (before commit — runtime only)
- [x] Build backend image with git SHA tag and `cloudrun-dev-latest`.
- [x] Push backend image tags.
- [x] Deploy backend dev using SHA-tagged image and record image tag, Cloud Run revision, and URL.
- [x] Source `scripts/dev-backend-env.sh` and run targeted smoke for:
  - successful `/api/v1/audit/lineage-search` query
  - empty lineage result
  - invalid direction or depth validation error
- [x] Create a complex dev fixture and verify business logic through deployed API:
  - Fixture ids: source `247HX7ab`, upstream/downstream nodes `akKFrap6`, `ZVrdOKvY`, `251729q4`, explicit revision node `ECUQl8na`, run `z646d`
  - Covered default dependency relation types excluding `revision_of`
  - Covered `depth=2` truncation and `depth=3` bounded cycle traversal
  - Covered `relation_types=derived_from` and `relation_types=revision_of`
  - Covered upstream multi-parent traversal and `direction=both`
- [x] Run relevant `scripts/api-guide-smoke.sh` lineage-search checks on dev.
  - Note: full script ended `21 passed, 1 failed` because root `/healthz` returned Cloud Run 404, a known non-business smoke issue; CYB-1098 lineage success check passed.

### Deploy record — CYB-1098
| Service | Image tag | Cloud Run revision | URL |
|---------|-----------|-------------------|-----|
| backend-dev | `cyber-databrew-backend:ed7886c-cyb1098` | `cyber-databrew-backend-dev-00166-tck` | `https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app` |

## PR
- [ ] PR template filled with Linear `CYB-1098`, OpenSpec change id, tests, and deploy evidence.
- [ ] Linear updated with commit/deploy summary after merge.
