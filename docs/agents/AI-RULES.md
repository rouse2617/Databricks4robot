# AI Agent Rules — cyber-databrew

This is the **single source of truth** for all AI agents working in this repository (Cursor, Codex, Claude Code, and others). Tool-specific entry files only **point here** — see [`TOOL-ENTRY.md`](TOOL-ENTRY.md).

**Users do not run a separate "setup" or "compliance" step.** When someone describes a task in natural language, you MUST apply these rules automatically — without asking "should I follow the project workflow?" **Never imply that deploy, OpenSpec, or hooks are Cursor-only.**

---

## Automatic behavior (every task)

On **every** user message that involves this repo, do the following **before** editing code:

1. **Silent environment check** — Follow `docs/agents/SETUP.md` in the background. Fix what you can (deps, skills, **repo-local commit-msg hook** via `git config core.hooksPath .githooks`). Only interrupt the user if a required credential is missing (e.g. `LINEAR_API_KEY`). OpenSpec is **in-repo Markdown only** — never ask to install OpenSpec CLI.
2. **Classify the task** — Bug fix, feature, hotfix, or docs/infra-only. Pick the path in `docs/agents/WORKFLOWS.md`.
3. **Linear** — Ensure a Linear Issue exists (`CYB-xxx` in this workspace). Create or link via MCP if the user did not provide one.
4. **OpenSpec (write artifacts, then stop)** — For any runtime change, create or use `openspec/changes/CYB-{id}-{slug}/` **before** writing application code. Follow [`spec-writing-skill.md`](spec-writing-skill.md) for artifact quality (proposal, tasks, spec delta; feature path also `design.md`). **Checkpoint:** When `proposal.md` + `tasks.md` (+ `design.md` if feature) are ready, **stop** and ask the user to confirm OpenSpec is OK (e.g. 「OpenSpec OK，继续」). **Do not** edit `backend/`, `Frontend/`, `sdk/`, or `dagster/` until they approve. Record their approval in `decisions.md` or a short Linear comment if useful.
5. **Branch** — Use `fix/CYB-{id}-*`, `feat/CYB-{id}-*`, or `hotfix/CYB-{id}-*` as appropriate (`DAT-*` accepted by CI for legacy). **Always branch from latest `dev`** (`git fetch origin dev && git checkout -b feat/CYB-{id}-… origin/dev`) so parallel CYB work does not conflict. Create the branch when starting OpenSpec or immediately after the OpenSpec checkpoint passes.
6. **After each code change** — Run verification at the **tier** matching diff scope (see [Verification tiers](#verification-tiers)); log non-obvious choices in `decisions.md` when required. **Any new or changed HTTP API** (route, handler, request/response, query param, status code) MUST complete [API contract sync](#api-contract-sync-mandatory) in the **same PR** as `backend/` — not a follow-up. **Bugs:** follow [`systematic-debugging`](skills/systematic-debugging/SKILL.md) before speculative fixes. **New interface methods:** when adding methods to a repository/usecase interface, ALWAYS update ALL test mocks (`*_test.go`) that implement that interface in the SAME commit — do not defer test mock updates.
7. **Before commit/push** — Follow [`deploy-before-commit.md`](deploy-before-commit.md) and [`deploy-verification.md`](deploy-verification.md) (canonical dev scripts in **§2.0**). **Dev is manual deploy** — `git push` does not roll Cloud Run; use local `deploy/cloudrun/*-dev.sh` or PR comment `/deploy-cloudrun-dev`. **If the diff touches `Frontend/`:** deploy frontend dev → Agent **must** run **Chrome DevTools MCP**. **Backend/sdk-only:** `source scripts/dev-backend-env.sh` + targeted smoke; `scripts/apply-migration-dev.sh` **before** backend deploy when schema changes.
8. **PR** — Fill `.github/pull_request_template.md` completely when opening a PR.
9. **Post-PR review loop (agent closes the loop)** — After opening a PR, actively check for review comments (`gh pr view <n> --json reviews,comments` and `gh api repos/:owner/:repo/pulls/<n>/comments` — `:owner`/`:repo` are auto-populated placeholders; `<n>` can be omitted when checking the PR of the current branch). Do **not** wait for the user to relay bot / human feedback. Read every unresolved comment. For each: (a) if it points at a real bug or gap, fix it — same PR if unmerged, follow-up PR if already merged (ask the user revert-vs-follow-up before acting); (b) if you disagree, reply on the PR with the reasoning; (c) if it needs the user's judgment (product decision, out-of-scope suggestion), summarize it back and ask. Report new PR numbers back to the user. **Never leave a bot's "critical" / "high-priority" finding uninvestigated.** Recheck after each push in case the reviewer runs again.

Do **not** ask the user to confirm that you will follow this workflow. Do **not** wait for them to say "按规范" or "prepare environment".

---

## What users typically say (you handle the rest)

| User says (examples) | You automatically do |
|----------------------|----------------------|
| "BatchDelete 删完不刷新" | Infer bug → Linear issue → OpenSpec change → `fix/CYB-*` branch → fix → verify → PR fields |
| "做 CYB-58 导出 API" | Load CYB-58 → feature path → OpenSpec (incl. design if feature) → `feat/CYB-58-*` → implement |
| "hotfix 线上 P0" | `hotfix-approved` label → skip OpenSpec gate only → still deploy verify → note backfill spec T+2 |
| "只改 README" | Docs-only path → no OpenSpec for runtime |

---

## Workflow constraints (mandatory)

- Runtime paths: `backend/`, `Frontend/`, `sdk/`, `dagster/` — see `docs/agents/spec-driven-workflow.md`
- Commit format: Conventional Commits — `type(scope): description`. **Subject (first line) must be entirely lower-case** — CI `commitlint` enforces `subject-case` (e.g. write `cel` / `api`, not `CEL` / `API`; acronyms in the body are fine).
- Full step tables: `docs/agents/WORKFLOWS.md`
- **New/changed HTTP API** → [API contract sync](#api-contract-sync-mandatory) (same PR, no exceptions except documented hotfix backfill)

## API contract sync (mandatory)

**Trigger:** You add or change anything exposed over HTTP — new `routes.go` registration, handler method, JSON body/query/path param, response shape, status code, or auth/idempotency header requirement.

**Do not** merge backend-only handler work and “document SDK later”. CYB-1014-style gaps (handler shipped, OpenAPI/api-guide/SDK empty) are **process failures**.

Sync these artifacts in the **same change / PR** (check off in `tasks.md`):

| # | File / area | Required when | What to update |
|---|-------------|---------------|----------------|
| 1 | [`api/openapi.yaml`](../../api/openapi.yaml) | Always | `paths`, `components/schemas`, parameters, request/response bodies, error envelope |
| 2 | [`docs/review/api-guide.md`](../../docs/review/api-guide.md) | Always | Section with `curl` examples, headers (`X-Databrew-Token`, `Idempotency-Key` if any), success + ≥1 error path, field validation notes |
| 3 | [`sdk/src/cyber_databrew_sdk/`](../../sdk/src/cyber_databrew_sdk/) | New/changed **public** REST surface | Resource client module (e.g. `customers.py`), methods mirroring api-guide; wire on [`client.py`](../../sdk/src/cyber_databrew_sdk/client.py) / [`__init__.py`](../../sdk/src/cyber_databrew_sdk/__init__.py) exports |
| 4 | [`sdk/tests/unit/`](../../sdk/tests/unit/) | SDK client added/changed | Unit tests for new client methods (mock HTTP) |
| 5 | [`scripts/api-guide-smoke.sh`](../../scripts/api-guide-smoke.sh) **or** `scripts/smoke-<feature>-dev.sh` | Always | At least happy path + one error path for **each new endpoint**; use `source scripts/dev-backend-env.sh` for dev |
| 6 | `backend/internal/handlers/*/*.go` | Handlers use Swagger generation elsewhere | `@Summary` / `@Router` / `@Param` blocks consistent with asset handlers (keep OpenAPI as source of truth if drift) |
| 7 | `openspec/changes/CYB-*/specs/*/spec.md` | Always (runtime feature) | Behavior delta (Given/When/Then) — **not** field-level API paste |
| 8 | `Frontend/src/api/` or feature hooks | UI calls the new API | Typed client / hook + types aligned with OpenAPI |

**Also sync when applicable (not HTTP, but same discipline):**

| Change | Also update |
|--------|-------------|
| `asset_events` event type / payload | `backend/schemas/events/*.json` + `registry.json` |
| Postgres DDL | `backend/migrations/` (approved) + `docs/review/sql.md` |
| Preview / gateway-only paths | `docs/review/api-guide.md` + OpenAPI if externally consumed |

**Verification before commit:** Tier **L** when OpenAPI or public API changes; run `cd sdk && uv run pytest tests/unit/` if SDK touched; run targeted smoke (`api-guide-smoke.sh` or feature script). PR template must list which rows above were updated.

**Out of scope declaration:** If an issue explicitly defers SDK or Frontend (e.g. “backend-only spike”), record it in `decisions.md` **and** Linear — still require rows **1, 2, 5, 7** minimum.

## Off-limits zones (require explicit approval to touch)

- `backend/internal/middleware/auth*`
- `backend/internal/outbox/`
- `backend/migrations/`
- `.env*` / any credential files
- `schemas/pg-phase0.sql`

If you must touch these, state it explicitly and expect a second reviewer. **`hotfix-approved` does not waive off-limits** — see [Rule precedence](#rule-precedence).

## Rule precedence

When rules appear to conflict, apply **the first matching row** (higher wins). Record the resolution in `openspec/changes/CYB-*/decisions.md` (or PR body for hotfix without a change dir).

| Priority | Rule | Examples |
|----------|------|----------|
| 1 | **User explicit instruction in the current message** | "只改 README" → docs-only path |
| 2 | **Safety & secrets** | Never commit credentials; never bypass auth on new paths |
| 3 | **Off-limits zones** | Hotfix touching `auth*` still needs explicit user approval + ≥2 reviewers |
| 4 | **Deploy-before-commit** | Runtime change → deploy dev + verify before commit (not waived by hotfix) |
| 5 | **OpenSpec / Linear traceability** | Runtime work needs CYB-xxx + change dir (except `hotfix-approved` skips **gate only**) |
| 6 | **CI hard gates** | commitlint, test-integration, openspec-gate (unless exempt label) |
| 7 | **Default workflow** | WORKFLOWS.md path for bug/feature/hotfix |

**Common conflicts:**

| Situation | Resolution |
|-----------|------------|
| Hotfix + must edit `auth*` | Proceed only after user confirms in chat; PR needs ≥2 reviewers; append `decisions.md` on backfill |
| User says "skip tests" | Run at least **Tier S** (lint/fmt); explain in `decisions.md` if skipping Tier M/L |
| `docs-only` but diff touches `backend/` | Not docs-only — full runtime path |
| Deploy gate vs "commit now" | Deploy verify first; commit only after user approves deploy |

## Verification tiers

Avoid running full `build` on every one-line fix. After each **accepted** code edit, run the **highest tier** that applies:

| Tier | When | Backend | Frontend | SDK |
|------|------|---------|----------|-----|
| **S — small** | ≤2 files, no router/handler/middleware/OpenAPI, no shared types | `make fmt && make vet` | `npm run lint` | `ruff check` on touched paths |
| **M — medium** | Default for most PRs; new/changed logic; >2 files or tests exist for package | Tier S + `go test` packages touched (`go test ./internal/foo/...`) | Tier S + `npm run test -- --run` (related tests if known) | Tier S + `pytest` for touched modules |
| **L — large** | Cross-module; UI routes; **any API contract sync**; build/config; before PR / deploy | Tier M + `go test ./...` | Tier M + `npm run build` | Tier M + full `pytest tests/unit/` |

**Always Tier L before:** opening PR, deploy verification, or touching off-limits-adjacent code.

**Upgrade triggers (bump one tier):** changed `go.mod` / `package.json` deps; renamed exported symbols; modified `App.tsx` routes or shared `components/common/*`; any edit under `openspec/specs/` (baseline behavior — treat as **Tier L**).

## API contract first (mandatory for new endpoints)

When implementing a **new HTTP endpoint** (not modifying an existing one), follow this order:

1. **Define the response shape** in `api/openapi.yaml` AND `Frontend/src/api/types.ts` (or the relevant API module) — BEFORE writing handler code.
2. **Verify shape alignment** — the backend handler's `c.JSON()` response MUST match the TypeScript type the frontend expects (e.g. `{items: [...], total: N}` vs bare array).
3. **Then implement** the handler, usecase, and frontend consumption.

**Why:** Mismatched response shapes (e.g. backend returns `[{...}]`, frontend expects `{items: [...]}`) cause silent failures that only surface during live testing. Defining the contract first prevents this class of bug entirely.

## Schema changes via Atlas migrations (mandatory)

**Golden rule: every schema change lands as a migration file in `backend/migrations/` FIRST and reaches dev/prod ONLY through PR merge → CI deploy-migrate. Local developers, local scripts, ad-hoc `psql` sessions, and any other path are NOT authorized to mutate the dev/prod schema.**

In practice: do not `psql` ALTER/CREATE/DROP/GRANT against dev/prod from a local terminal, a debug script, a notebook, or any other side channel. (This is exactly how the pre-2026-07 `node_runs`/`pipeline_definitions` drift and the ad-hoc `api_keys` table happened.) Manual DDL creates untracked drift — the change is not in the repo, does not reach other environments through the normal flow, and makes dev and prod diverge.

**Hotfix exception:** production emergencies use the standard `hotfix-approved` label flow — see [Workflow constraints](#workflow-constraints) and `docs/agents/WORKFLOWS.md` — which is the one authorized shortcut, still tracked through PR → CI, still requiring a second reviewer on off-limits changes.

**Migrations are hand-written SQL.** The GORM structs in `backend/internal/dbschema/` are reference/ORM only — they are **not** the migration source, and `atlas migrate diff --env gorm` is **not** used: GORM cannot express this schema's CHECK constraints, triggers, functions, partitions, trigram indexes, or GENERATED columns, and diffing against it emits destructive output. Write those objects by hand in the migration.

**Workflow to add/change a table or column:**

1. Create `backend/migrations/<YYYYMMDDHHMMSS>_<name>.sql` (timestamp prefix from `date +%Y%m%d%H%M%S`, strictly later than the newest existing file). Use plain `ALTER`/`CREATE` — no `IF NOT EXISTS` for normal forward migrations. A new Postgres extension needs an explicit `CREATE EXTENSION IF NOT EXISTS …` (Atlas does not emit these — the `pg_trgm` gap bit us once).
2. Update the Go struct that reads/writes the table.
3. `cd backend && make db-migrate-hash` (recompute `atlas.sum`).
4. `atlas migrate validate --env migrate --dev-url "docker://postgres/17/dev?search_path=public"` (fresh-PG17 replay of every migration + checksum) — this is what CI's `db-migrate-lint` runs.
5. PR → merge to `dev` → `deploy-migrate` applies to live dev; prod gets it on `tag → main → deploy-prod`. Both envs are baselined, so deploy runs a plain `atlas migrate apply` (pending only).

Never hand-edit `atlas.sum` or an already-applied migration file. Adopting a pre-existing DB into Atlas is a one-off `atlas migrate apply --baseline <version>` by hand, not a deploy step. Full runbook: [`docs/atlas-migrations.md`](../atlas-migrations.md).

## Migration check before deploy (mandatory)

When a PR includes a new migration file (`backend/migrations/*.sql`):

1. **Apply migration to dev** using `bash scripts/apply-migration-dev.sh "$(pwd)/backend/migrations/NNN_name.sql"` BEFORE deploying the backend image.
2. **Verify migration applied** — run a smoke query that touches the new columns/tables.

**Why:** Deploying code that references new columns before the migration is applied causes runtime 500 errors on every request that touches those columns.

## Decision log (audit trail)

Agents MUST append to `openspec/changes/CYB-{id}-{slug}/decisions.md` when:

- Resolving a [rule precedence](#rule-precedence) conflict
- Touching or requesting exception to off-limits
- Choosing verification Tier S when M would normally apply (user pressure or timebox)
- Skipping or deferring deploy verification (only with user-written approval in chat)
- **Chrome DevTools MCP unavailable** while diff touches `Frontend/` (document blocker; user may assist)

**Hotfix without change dir:** put the same entries in the PR description under `## Agent decisions`.

Format (append-only):

```markdown
## YYYY-MM-DD — Short title
- **Context**: …
- **Decision**: …
- **Alternatives**: …
- **Rationale**: …
```

## Rule maintenance

| What | Who | How |
|------|-----|-----|
| `docs/agents/**`, `openspec/config.yaml`, entry pointers | Any engineer; **review required** | PR titled `docs(agents): …` or bundled in feature PR when workflow changes; sync `config.yaml` `context`/`verification` with [`domain.md`](domain.md) when stack/modules change |
| `openspec/specs/**` (baseline behavior) | Feature owner + reviewer | Merged via Feature archive step |
| `.cursor/rules/*.mdc` | Maintainers only | Thin pointers to `docs/agents/` — no duplicate policy text |
| CI gates (`openspec-gate`, `commitlint`) | Platform / repo admins | Change only with team notice in PR |

**Principles:** one source of truth (`docs/agents/`); tool entry files stay pointers; user-facing README summarizes, links here for detail.

## Linear integration

- Every deliverable change MUST have a Linear Issue (create via MCP if missing)
- PR body MUST contain Linear ID (`CYB-xxx`) and OpenSpec change-id
- Update Linear state to Done after merge, include commit hash

## Skills and tools

- Trigger skills per scenario without being asked — see `docs/agents/SKILLS.md`
- MCP rules: `docs/agents/MCP-POLICY.md`

## Verification commands (reference)

Full commands by tier — see [Verification tiers](#verification-tiers).

| Module | Tier S | Tier M | Tier L |
|--------|--------|--------|--------|
| Backend | `make fmt && make vet` | + `go test ./path/to/pkg/...` | + `go test ./...` |
| Frontend | `npm run lint` | + `npm run test -- --run` | + `npm run build` |
| SDK | `uv run ruff check src/` | + `pytest` touched | + `pytest tests/unit/` |

## Local dev backend modes

Use one unified local frontend entry point and choose the backend mode explicitly:

| Mode | Command | Backend target | Use for |
|------|---------|----------------|---------|
| Cloud Run dev | `bash scripts/dev-local.sh --cloudrun` | `cyber-databrew-backend-dev` resolved by `scripts/dev-backend-env.sh` | Canonical dev backend parity and deploy-verification-style checks |
| Shared Pod | `bash scripts/dev-local.sh --shared` | shared GKE service `cyber-databrew-backend` | Local frontend work against the stable shared dev pod |
| Fast preview Pod | `bash scripts/dev-local.sh --deploy` or `--preview-id <id>` | isolated GKE preview backend under `/preview/<id>/api` | Backend changes that need quick frontend integration without changing Cloud Run dev |

Do not mix these up:

- **Cloud Run dev** is the canonical shared dev deploy target. Use `source scripts/dev-backend-env.sh` or `scripts/dev-local.sh --cloudrun`; do not hand-copy backend URLs.
- **Fast preview Pod** is commit/HEAD-based and isolated. It is appropriate for quick backend/frontend pairing, but it does not replace Cloud Run dev deploy verification before commit/PR.
- **Local frontend** should normally stay local in all three modes; only deploy frontend dev when doing required deployed-revision UI verification.

## Development pitfalls (lessons learned)

Field incidents that should inform future development. Add to this section when a preventable mistake repeats.

### P1. Verify diff, not commit message
After editing code (especially tab-heavy Go like switch/struct blocks), run `git diff` before committing. The `default` branch in a `switch` can be silently missing if the edit tool fails on tab indentation — and the commit message may claim it's there.

**Check:** `git diff --stat` and spot-check the actual line changes, not just "build passes".

### P2. Use valid test payloads against DB constraints
When testing API behavior on dev, ensure the payload satisfies DB CHECK constraints (e.g. `chk_lifecycle_state`). An invalid value like `"active"` produces a DB error that masks the real issue (e.g. validator not rejecting unknown types).

**Check:** Before curl/writing a test, look up valid values from the model layer or migration — don't guess.

### P3. Confirm SHA before docker build
`docker build` uses the working tree, but the SHA tag should match the current HEAD. An incorrect tag (e.g. stale `93d2bc7` instead of current `c8c175c`) means the deployed Cloud Run revision runs stale code.

**Check:** `echo "HEAD: $(git rev-parse --short HEAD)"` before `docker build`.

### P4. Worktree-aware PR merge
When the base branch is checked out in another worktree, `gh pr merge` fails locally. Use `gh pr merge --auto` to let GitHub's API handle the merge, avoiding the worktree conflict entirely.

**Check:** If you get "already used by worktree", retry with `--auto`.

### P5. Deploy verification must assert the specific change
Generic L1 smoke (health + list) doesn't prove the behavioral change works. After deploying, immediately curl with a known-bad payload that should hit the new code path.

**Check:** For each runtime change, write one curl command (happy path + one error path) that proves the new behavior is active on the deployed revision. Include the command and expected response in the PR.

### P6. Python edit scripts: verify line count
When using Python scripts as an edit-tool workaround (Go tab indentation), the off-by-one in line deletion is silent — it can leave a stale line or omit a needed one. After the script runs, read the affected function top-to-bottom.

**Check:** `sed -n 'start,endp' file.go` on the edited region and visually confirm.

### P7. Files must end with exactly one newline (no trailing blank lines)
The `end-of-file-fixer` pre-commit hook fails when a file has trailing blank lines after the final newline. Python's `__init__.py`, Go files, and any tracked file are subject to this check. This has failed CI repeatedly.

**Check:** `pre-commit run end-of-file-fixer --all-files` before committing, or manually verify the file ends with a single `\n` (no extra blank lines). When using the Edit/Write tools, ensure the last line of content is not followed by an empty line.

### P8. Argo Server lives in K8s, not Cloud Run — use ARGO_SERVER_URL

**UPDATE (2026-06-24):** K8s auth is now Workload Identity. See above — do NOT use `K8S_BEARER_TOKEN`.

### P9. Cloud Run dev deploy command

```bash
gcloud run deploy cyber-databrew-backend-dev \
  --image=us-central1-docker.pkg.dev/green-valley-442103/cyber-databrew-images/cyber-databrew-backend:dev-latest \
  --region=us-central1 --project=green-valley-442103 \
  --service-account=cyber-databrew-dev@green-valley-442103.iam.gserviceaccount.com
```
**Must include `--service-account`** — omitting it reverts to Compute Engine default SA and breaks K8s WI auth.

### P10. grace_video asset type (UUID asset IDs)

Since 2026-06-24, DataBrew supports UUID-format asset IDs alongside the original 8-char format. The `grace_video` asset type:
- Uses UUID as `asset_id` (Grace segmentation_id)
- Does NOT require `mcap_file_id` (schema bypasses it)
- DB constraints updated: `assets_asset_id_check`, `chk_mcap_file_required`, etc.
- Migration: `backend/migrations/060_grace_video_asset_id.sql`
- Trigger: `trg_nullify_empty_mcap` converts empty mcap_file_id to NULL
- When creating grace_video assets, set `asset_type: "grace_video"` and skip `mcap_file_id`
Argo Workflows API server runs inside the dev K8s cluster (e.g. `http://10.2.1.211:2746`), NOT on Cloud Run. The Cloud Run service `cyber-databrew-pipeline-ui-dev` is a separate UI proxy, not the Argo API.

**Symptoms of confusion:** `ARGO_BASE_URL` pointing at the Cloud Run pipeline-ui returns `{"code":"UNAUTHORIZED","message":"invalid token: ... unexpected signing method: RS256"}` because that proxy uses different auth (SSO/OIDC) and can't verify K8s SA tokens.

**Check:** On Cloud Run dev backend, `ARGO_SERVER_URL` should be the K8s in-cluster IP (e.g. `http://10.2.1.211:2746`) and `ARGO_WORKFLOWS_NAMESPACE=cyber-databrew-dev`. K8s ServiceAccount tokens (RS256) work because the K8s-based Argo server verifies them via TokenReview. Note that `deploy/cloudrun/backend-dev.sh` uses `--env-vars-file`, which overwrites revision environment variables on deploy. Therefore, do not hand-edit Cloud Run Console env vars for backend dev; keep canonical values in the script or pass explicit overrides. **K8s auth for Cloud Run is Workload Identity (since 2026-06-24).** The old static `K8S_BEARER_TOKEN` approach is deprecated — tokens expired every 2 days. Current env:
```
K8S_USE_METADATA_TOKEN=true       ← WI enabled, token auto-refreshes
K8S_AUDIENCE=https://34.59.48.233
K8S_API_ENDPOINT=https://34.59.48.233
K8S_CA_DATA  → Secret: cyber-databrew-dev-k8s-ca-data
```
**Do NOT remove `K8S_USE_METADATA_TOKEN` or change the SA when redeploying.** SA must be `cyber-databrew-dev@green-valley-442103.iam.gserviceaccount.com`. K8s SA `cyber-databrew-backend-argo` has WI binding to this GCP SA. Losing these causes `create runtime config projection: Unauthorized` on every pipeline run.

### Execution targets (Argo namespaces)

> Adding a namespace/target to an **already-onboarded** cluster is below. Onboarding a **brand-new cluster** (new K8s API + `clusters` row) → [`cluster-onboarding.md`](cluster-onboarding.md).

DataBrew pipeline runs dispatch to K8s namespaces via execution targets. Current targets:

| Target ID | Namespace | Argo Controller |
|-----------|-----------|-----------------|
| `default` | `cyber-databrew-dev` | `argo-workflows-workflow-controller` (shared) |
| `a03ad932-...` | `video-proc-dev` | `argo-workflows-workflow-controller-video-proc-dev` |
| `video-proc-prod` | `video-proc-prod` | `argo-workflows-workflow-controller-video-proc-prod` (2026-06-25) |

**video-proc-dev** (16d ago):
- Deploy: `argo-workflows-workflow-controller-video-proc-dev`
- SA: `argo-workflow`, `argo-workflows-workflow-controller`, `workflow-runner`
- Role: `argo-workflows-workflow`, `argo-workflows-workflow-controller`, `cyber-databrew-backend-runtime-config`, `cyber-databrew-resource-capacity-reader`

**video-proc-prod** (2026-06-25):
- Deploy: `argo-workflows-workflow-controller-video-proc-prod`
- SA: `argo-workflows-workflow-controller`, `workflow-runner`
- Role: `argo-workflows-workflow`, `argo-workflows-workflow-controller`, `cyber-databrew-backend-runtime-config`
- ConfigMap: `argo-workflows-workflow-controller-configmap` (copied from `cyber-databrew-dev`)

**To add a new execution target namespace**, replicate these from an existing one (e.g. `video-proc-dev`):
1. SA `argo-workflows-workflow-controller` + SA `workflow-runner`
2. Role `argo-workflows-workflow` + Role `argo-workflows-workflow-controller` + Role `cyber-databrew-backend-runtime-config`
3. RoleBinding for each Role → corresponding SA
4. Argo Controller Deployment + ConfigMap (namespaced mode, `--namespaced` flag)
5. DB: `INSERT INTO execution_targets ...`

**Do NOT create or modify K8s Secrets** without explicit approval. Secret content must come from the user.

## What NOT to do

- Do NOT ask users to run a ritual prompt like "prepare environment per project standards"
- Do NOT write runtime code before OpenSpec change exists (except `hotfix-approved` with documented backfill)
- Do NOT ship new/changed HTTP handlers without [API contract sync](#api-contract-sync-mandatory) (OpenAPI + api-guide + smoke minimum)
- Do NOT commit without deployment verification (deploy-before-commit rule)
- Do NOT ask the user to「浏览器点一遍」when the diff touches `Frontend/` and Chrome DevTools MCP is available — run MCP yourself on dev
- Do NOT introduce new dependencies without declaring them
- Do NOT bypass auth checks on new paths
- Do NOT refactor or clean up unrelated code
