# Tasks: CYB-1215 Grace→DataBrew rename

## Overview

**Goal**: Rename `X-Grace-Token` → `X-Databrew-Token` (header, env var, Go names, cookie, query param) across every layer.

**Branch**: `feat/CYB-1215-grace-to-databrew` (from `dev`)

**Off-limits**: `backend/internal/middleware/auth.go` — explicit approval obtained from user (they asked to "先完成吧").

**Strategy**: Single PR with code changes only. Env var updates in Cloud Run / K8s are separate deploy steps.

## Task list

### ⬜ T1. OpenSpec approval
- [x] Create `openspec/changes/CYB-1215-grace-to-databrew/`
- [x] Write `proposal.md`
- [ ] Write `tasks.md` ← this file
- [ ] User confirms "OpenSpec OK"

### ⬜ T2. Create branch
- `git fetch origin dev && git checkout -b feat/CYB-1215-grace-to-databrew origin/dev`

### ⬜ T3. Rename Go backend references

#### core auth + config
- `backend/internal/middleware/auth.go` — `GetHeader("X-Grace-Token")` → `"X-Databrew-Token"`
- `backend/internal/config/config.go` — struct field `GraceToken` → `DatabrewToken`, `getenv("GRACE_TOKEN", ...)` → `getenv("DATABREW_TOKEN", ...)`
- `backend/routes/routes.go` — `cfg.GraceToken` → `cfg.DatabrewToken`, comment references

#### handlers (Swagger annotations)
- All `// @Security GraceToken` → `// @Security DatabrewToken` in:
  - `backend/internal/handlers/asset/handler.go`
  - `backend/internal/handlers/asset/algo_handler.go`
  - `backend/internal/handlers/asset/mcap_locator.go`
  - `backend/internal/handlers/audit/handler.go`
  - `backend/internal/handlers/customer/handler.go`
  - `backend/internal/handlers/delivery/handler.go`
  - `backend/internal/handlers/algorun/handler.go`

#### Swagger generated docs
- `backend/docs/swagger/swagger.yaml` — `GraceToken` → `DatabrewToken`, `X-Grace-Token` → `X-Databrew-Token`
- `backend/docs/swagger/docs.go` — same replacements (re-run `swag init` instead of manual edit)
- If `swag` CLI unavailable, `sed` + manual fix on the generated files.

### ⬜ T4. Rename SDK
- `sdk/src/asset_sdk/client.py` — `"X-Grace-Token"` → `"X-Databrew-Token"`, `GRACE_TOKEN` → `DATABREW_TOKEN` in env and comments
- `sdk/README.md` — `GRACE_TOKEN` → `DATABREW_TOKEN`

### ⬜ T5. Rename OpenAPI spec
- `api/openapi.yaml` — security scheme name `GraceToken` → `DatabrewToken`, header name `X-Grace-Token` → `X-Databrew-Token`
- Also the global `security: - GraceToken: []` → `- DatabrewToken: []`

### ⬜ T6. Rename mcap-preview service
- `services/mcap-preview/internal/server/server.go` — `extractGraceToken` → `extractDatabrewToken`, `GraceTokenPassthrough` → `DatabrewTokenPassthrough`, `"X-Grace-Token"` → `"X-Databrew-Token"`, `"grace_token"` → `"databrew_token"`, `"grace_session"` → `"databrew_session"`
- `services/mcap-preview/internal/server/segment.go` — same references
- `services/mcap-preview/internal/server/server_test.go` — test constants and header sets
- `services/mcap-preview/internal/server/segment_test.go` — same
- `services/mcap-preview/cmd/server/main.go` — `GraceTokenPassthrough` → `DatabrewTokenPassthrough`, `GRACE_TOKEN_PASSTHROUGH` → `DATABREW_TOKEN_PASSTHROUGH`

### ⬜ T7. Rename deploy configs
- `deploy/k8s/frontend/nginx-config.yaml` — `X-Grace-Token` → `X-Databrew-Token`, `grace_session` → `databrew_session`
- `deploy/cloudrun/frontend-nginx.conf` — same
- `Frontend/nginx.conf` — same
- `deploy/k8s/base/secret.example.yaml` — `GRACE_TOKEN` → `DATABREW_TOKEN`
- `deploy/k8s/mcap-preview/configmap.yaml` — `GRACE_TOKEN_PASSTHROUGH` → `DATABREW_TOKEN_PASSTHROUGH`
- `deploy/local/docker-compose.yml` — `GRACE_TOKEN` → `DATABREW_TOKEN`
- `deploy/k8s/jobs/seed-rich-500-pod.yaml` — `GRACE_TOKEN` → `DATABREW_TOKEN`
- `deploy/k8s/jobs/api-guide-smoke-job.yaml` — `GRACE_TOKEN` → `DATABREW_TOKEN`
- `.github/workflows/nightly-es-pg-audit.yml` — `GRACE_TOKEN` → `DATABREW_TOKEN`
- `deploy/cloudrun/backend-dev.sh` — `GRACE_TOKEN_OVERRIDE` → `DATABREW_TOKEN_OVERRIDE` (wait — this is an env var name on Cloud Run, could break deployed envs. Keep as is for now, deploy team updates separately.)
  - Actually, this is a LOCAL deploy script that sets local env vars. Safe to rename.

Decision: rename env var names in deploy scripts and local configs. Do NOT rename the K8s Secret key names (GRACE_TOKEN key in the Secret object) — those are infrastructure state. Update only the env var NAME that Go code reads from `os.Getenv`. The actual env var key value in Cloud Run is set independently.

Wait, let me reconsider. The Go code reads `os.Getenv("GRACE_TOKEN")` which must match the env var name set in the Cloud Run service. If I rename the Go code to read `os.Getenv("DATABREW_TOKEN")`, the Cloud Run env must also be updated. This is a coordinated deploy:

1. Deploy this PR (code reads `DATABREW_TOKEN`)
2. Update Cloud Run env vars to set `DATABREW_TOKEN=...` (and keep `GRACE_TOKEN=...` as fallback during roll-forward)
3. Remove `GRACE_TOKEN` from Cloud Run after the deploy is confirmed

Actually the cleanest approach: rename both the env var name AND add backward compatibility. The Go config can read `DATABREW_TOKEN` first, fall back to `GRACE_TOKEN`:
```go
DatabrewToken: getenv("DATABREW_TOKEN", getenv("GRACE_TOKEN", "dev-token"))
```
This way the code deploys cleanly even if Cloud Run still has `GRACE_TOKEN` set.

Updated task for T3:

### ⬜ T3. Rename Go backend references (revised)

- `backend/internal/config/config.go` — `GraceToken` → `DatabrewToken`, env fallback: `getenv("DATABREW_TOKEN", getenvFallback("GRACE_TOKEN", "dev-token"))`
Note: `getenvFallback` doesn't exist. Better: implement a two-level default inline.

Actually, the simplest approach that doesn't require infrastructure coordination:

```go
DatabrewToken: getenv("DATABREW_TOKEN", "dev-token")
```

And in the deploy scripts/nginx configs, change the env var. For Cloud Run, update env vars as a separate step. The rename is clean — one PR, one env var per service.

For K8s (not Cloud Run): the `secret.example.yaml` key name doesn't affect the runtime — what matters is the env var name set in the pod template. The pod templates reference `secretKeyRef.name: grace-token` — that key name in the secret object doesn't change.

Let me simplify: rename the env var name that Go/Python reads. The actual provisioning of that env var value (Cloud Run UI, K8s Secret, etc.) is handled separately.

So:
- Go: `getenv("DATABREW_TOKEN", "dev-token")` 
- Python: `os.environ.get("DATABREW_TOKEN", "")`
- Deploy scripts: `DATABREW_TOKEN=...`
- K8s pod templates: env var name in containers
- K8s Secret example: `DATABREW_TOKEN`
- Docker compose: `DATABREW_TOKEN`

For the CLoud Run env vars — those are set in the GCP console / `gcloud run deploy --update-env-vars`, NOT in the repo. The repo deploy scripts reference the env var name but don't set its value. We document the required env change.

### ⬜ T8. Rename frontend references
- `Frontend/src/hooks/assets/useAssetPreview.ts` — comment `X-Grace-Token` → `X-Databrew-Token`, `grace_token` → `databrew_token` (query param)
- `Frontend/src/hooks/assets/useAssetPreview.test.ts` — `grace_token` → `databrew_token`
- `Frontend/src/pages/LoginPage.tsx` — `GRACE_TOKEN` → `DATABREW_TOKEN` (UI label)
- `Frontend/e2e/assets-happy-path.spec.ts` — `grace_token` → `databrew_token` (localStorage key)
- `Frontend/README.md` — `X-Grace-Token` → `X-Databrew-Token`

### ⬜ T9. Rename scripts
- All shell scripts with `-H "X-Grace-Token"` → `-H "X-Databrew-Token"`
- All Python scripts with `"X-Grace-Token"` in headers → `"X-Databrew-Token"`
- All `GRACE_TOKEN` env var refs → `DATABREW_TOKEN`
- `scripts/dev-backend-env.sh` — `GRACE_TOKEN` → `DATABREW_TOKEN`
- `scripts/api-guide-smoke.sh` — `X-Grace-Token` → `X-Databrew-Token`, `GRACE_TOKEN` → `DATABREW_TOKEN`

### ⬜ T10. Rename docs
- `docs/review/api-guide.md` — bulk find-replace `X-Grace-Token` → `X-Databrew-Token`, `GRACE_TOKEN` → `DATABREW_TOKEN` (heaviest doc, ~80 occurrences)
- `docs/agents/AI-RULES.md` — line 55 reference
- `docs/agents/SETUP.md` — `GRACE_TOKEN`
- `docs/agents/deploy-verification.md` — `GRACE_TOKEN`, `X-Grace-Token`
- Other docs with scattered references

### ⬜ T11. Rename mcap-preview README
- `services/mcap-preview/README.md` — `GRACE_TOKEN_PASSTHROUGH` → `DATABREW_TOKEN_PASSTHROUGH`, `X-Grace-Token` → `X-Databrew-Token`

### ⬜ T12. Rename k6 loadtest scripts
- `deploy/local/loadtest/k6-write-smoke.js`
- `deploy/local/loadtest/k6-read-mix.js`
- `deploy/local/loadtest/run-ab-mix.sh`
- `deploy/local/loadtest/README.md`

### ⬜ T13. Verification
- `make fmt && make vet`
- `go test ./...` (backend)
- `ruff check sdk/src/`
- `npm run lint` (frontend)
- Verify `X-Grace-Token` no longer exists in codebase (non-worktree, non-vendor)

### ⬜ T14. Commit and PR
- Conventional commit: `refactor(api): rename X-Grace-Token to X-Databrew-Token across all layers (CYB-1215)`
- Update Linear state

## Verification

Required checks before PR:

1. `go test ./...` — all backend tests pass
2. `make fmt && make vet` — no formatting/lint issues
3. `git grep "X-Grace-Token" | grep -v ".git/" | grep -v ".claude/worktrees/" | grep -v vendor/` — confirms zero remaining old header strings
4. `git diff --stat` — confirm all expected files touched
5. Spot-check Go middleware reads `X-Databrew-Token`
6. Spot-check OpenAPI spec has `X-Databrew-Token` security scheme

Note: tier L verification (deploy dev) cannot be done for a single env var rename until Cloud Run env vars are also updated. Code-level correctness is sufficient for this PR.
