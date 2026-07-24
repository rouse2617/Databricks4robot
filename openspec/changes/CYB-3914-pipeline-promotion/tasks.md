# Tasks

## 0. Checkpoints and approvals

- [x] 0.1 [docs] Replace `CYB-TBD` in branch/change paths with the user-provided Linear ID before commit or PR. — Renamed `feat/CYB-TBD-pipeline-promotion` → `feat/CYB-3914-pipeline-promotion` and `openspec/changes/CYB-TBD-pipeline-promotion/` → `openspec/changes/CYB-3914-pipeline-promotion/` after Linear `CYB-3914` was created on 2026-07-24.
- [x] 0.2 [docs] Record user approval of this OpenSpec in `decisions.md` before runtime edits. — See decisions.md `2026-07-23 — OpenSpec and migration implementation approved` (user "ok，开发吧").
- [x] 0.3 [migration] Obtain explicit user approval before touching `backend/migrations/`; require a second reviewer. — Approved 2026-07-23 (see above); second reviewer still required at PR time.
- [x] 0.4 [security] If implementation must touch `backend/internal/middleware/auth*`, stop for separate explicit approval and require a second reviewer. — Not triggered: `git diff --name-only` and untracked files in this worktree touch no `middleware/auth*` paths.

## 1. API contract first

- [ ] 1.1 [api/openapi.yaml] Define promotion plan, execute, internal validate/import, versioned update, mappings, blockers, audit result, error statuses, `Idempotency-Key`, and `baseVersion`.
- [ ] 1.2 [Frontend/src/api] Define matching TypeScript request/response types before handler implementation.
- [ ] 1.3 [docs/review/api-guide.md] Add admin/service authentication, curl examples, validation rules, success and error paths.
- [ ] 1.4 [sdk/src/cyber_databrew_sdk] Add/update public pipeline promotion and version-save client methods and exports.
- [ ] 1.5 [sdk/tests/unit] Cover request paths, bodies, headers, and error propagation for SDK changes.
- [ ] 1.6 [scripts/smoke-pipeline-promotion-dev.sh] Add happy and error smoke coverage for every new public endpoint without writing real prod.

## 2. Pipeline identity and bundle construction

- [ ] 2.1 [backend/internal/transpiler] Preserve `componentId`, `releaseId`, and `componentVersionLabel` through JSON normalize/save/load. Covers “Release-backed component is bundled”.
- [ ] 2.2 [backend/internal/usecase/pipeline] Build an exact normalized bundle from the selected saved version and immutable runtime snapshot. Covers “Ready promotion plan” and “Release-backed component is bundled”.
- [ ] 2.3 [backend/internal/usecase/pipeline] Resolve legacy nodes only on one exact digest match and emit a warning. Covers “Legacy component has one digest match”.
- [ ] 2.4 [backend/internal/usecase/pipeline] Block zero/multiple legacy matches. Covers “Legacy component is ambiguous”.
- [ ] 2.5 [backend/internal/usecase/pipeline] Classify config/env entries and exclude/block sensitive plaintext. Covers “Sensitive plaintext is detected”.
- [ ] 2.6 [backend/internal/usecase/pipeline] Reject missing resource mappings before execution. Covers “Missing target mapping”.
- [ ] 2.7 [backend/internal/usecase/pipeline tests] Add fixture/golden tests proving normalization never drops release identity or changes digests.

## 3. Fixed transport and authorization

- [ ] 3.1 [backend/internal/config] Add fixed prod origin, audience/caller identity, timeouts, payload bounds, and dedicated fallback Secret configuration.
- [ ] 3.2 [backend/internal/handlers/pipeline] Require admin role for plan/execute and reject non-admin before target calls. Covers “Non-admin requests a plan”.
- [ ] 3.3 [backend/internal/handlers/pipeline promotion client] Reject request-provided URLs/unsupported targets and redirects. Covers “Caller supplies a target URL”.
- [ ] 3.4 [backend/internal/handlers/pipeline promotion auth] Validate dedicated service identity without reusing component-release credentials. Covers “Known dev service calls prod” and “Internal caller authentication fails”.
- [ ] 3.5 [backend/routes/routes.go] Register public admin and internal service endpoints with distinct authorization.

## 4. Target validation, import, and audit

- [ ] 4.1 [backend/migrations] After explicit approval, add hand-written Atlas migration and checksum for `pipeline_promotions`; validate fresh Postgres 17 replay.
- [ ] 4.2 [backend/internal/models and repository] Add target-authoritative promotion audit/idempotency model and repository with unique-key conflict semantics.
- [ ] 4.3 [backend/internal/usecase/pipeline promotion] Validate/rewrite mapped prod resource IDs without copying values. Covers “Logical resource has a valid prod mapping”.
- [ ] 4.4 [backend/internal/usecase/pipeline promotion] Return sanitized blockers for missing/unready/unauthorized target resources. Covers “Target resource is unavailable”.
- [ ] 4.5 [backend/internal/usecase/pipeline promotion] Validate digest pull readiness. Covers “Image digest is not pullable in prod”.
- [ ] 4.6 [backend/internal/usecase/pipeline promotion] Reject component/release-label collisions with different digests. Covers “Target release label conflicts by digest”.
- [ ] 4.7 [backend/internal/usecase/pipeline promotion] Transactionally import safe dependencies, allocate target version, and complete audit. Covers “First successful import”.
- [ ] 4.8 [backend/internal/usecase/pipeline promotion] Return the original result for same key/digest. Covers “Identical retry”.
- [ ] 4.9 [backend/internal/usecase/pipeline promotion] Return `409` for same key/different digest. Covers “Idempotency key is reused for different content”.
- [ ] 4.10 [backend/internal/usecase/pipeline promotion] Roll back all writes and sanitize failures. Covers “Import fails inside the transaction”.
- [ ] 4.11 [backend/internal/usecase/pipeline promotion] Recompute target state and reject stale plan digests. Covers “Accepted plan became stale”.

## 5. Prod versioned editing

- [ ] 5.1 [backend/internal/handlers/pipeline and usecase] Replace the update stub with append-only save using `baseVersion`. Covers “Admin saves a prod edit”.
- [ ] 5.2 [backend/internal/handlers/pipeline and usecase] Enforce admin-only prod edit/delete/activate. Covers “Non-admin edits prod”.
- [ ] 5.3 [backend/internal/repository/pipeline] Add atomic next-version/stale-base behavior. Covers “Admin edit is stale”.
- [ ] 5.4 [backend/internal/usecase/pipeline tests] Prove older versions and run references remain unchanged. Covers “Historical run is inspected”.
- [ ] 5.5 [backend/internal/handlers/pipeline] Role-gate and audit active-version changes. Covers “Admin activates a prod version”.

## 6. Frontend workflow

- [ ] 6.1 [Frontend/src/components/pipeline/DeployPanel.tsx] Add review → mapping → confirm promotion flow and ready/blocker/error states.
- [ ] 6.2 [Frontend/src/features/pipeline-designer] Send the selected saved version, plan digest, mappings, and a generated idempotency key. Covers “Existing promote action is used”.
- [ ] 6.3 [Frontend/src/features/pipeline-designer] Prevent execution without a current ready plan and surface stale-plan refresh. Covers “Promote is called without accepted plan”.
- [ ] 6.4 [Frontend pipeline list/editor] Show prod edit/activate controls only to admins; ordinary users retain view/run behavior.
- [ ] 6.5 [Frontend/src/components/pipeline/VersionHistoryDrawer.tsx] Display promoted/admin-created versions and retain historical diff/navigation.
- [ ] 6.6 [Frontend tests] Cover admin/non-admin visibility, mappings, retry-safe submit, blockers, stale plan, and versioned save.

## 7. Deployment configuration

- [ ] 7.1 [deploy/cloudrun/backend-dev.sh] Bind fixed prod target/service-auth settings without committing credentials.
- [ ] 7.2 [.github/workflows/deploy-prod.yml] Bind prod importer audience/caller or dedicated promotion Secret while preserving all existing `--set-secrets` entries.
- [ ] 7.3 [docs/review] Document service identity, Artifact Registry reader prerequisites, target catalog mapping, rotation, and rollback.
- [ ] 7.4 [operations] Confirm prod catalog coverage (including known delivery/storage differences) and node image-pull permission before enabling the UI action.

## 8. Verification and acceptance

- [ ] 8.1 [backend tests] Run Tier L: format/vet, targeted promotion tests, and `go test ./...`.
- [ ] 8.2 [Frontend tests] Run lint, related Vitest tests, and production build.
- [ ] 8.3 [sdk tests] Run Ruff and full SDK unit tests.
- [ ] 8.4 [migration] Apply the approved migration to dev through `scripts/apply-migration-dev.sh` before backend deploy and verify the table via API behavior.
- [ ] 8.5 [deploy] Deploy backend dev, record image/revision/URL, and run plan + importer-local/disposable happy/error/idempotency/rollback smoke.
- [ ] 8.6 [deploy] Deploy frontend dev, record revision, and use Chrome DevTools MCP for admin flow, non-admin state, console/network, screenshots, and adjacent pipeline regression.
- [ ] 8.7 [security] Verify arbitrary targets, invalid service identity, sensitive config, digest conflict, and non-admin writes are rejected with sanitized responses.
- [ ] 8.8 [performance] Measure plan/import against the stated 100-node SLO and record cold-start exclusions.
- [ ] 8.9 [prod safety] Do not execute a real prod write during routine dev verification; request separate operational approval for one post-deploy canary promotion.

## API contract synchronization checklist

- [ ] `api/openapi.yaml`
- [ ] `docs/review/api-guide.md`
- [ ] `sdk/src/cyber_databrew_sdk/`
- [ ] `sdk/tests/unit/`
- [ ] `scripts/api-guide-smoke.sh` or `scripts/smoke-pipeline-promotion-dev.sh`
- [ ] Handler Swagger annotations where present
- [ ] `openspec/changes/CYB-*/specs/*/spec.md`
- [ ] `Frontend/src/api/` typed client and hooks
