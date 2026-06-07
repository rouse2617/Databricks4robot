# Deploy-before-commit gate (REQUIRED)

Applies to **all AI agents** (Cursor, Codex, Claude Code, etc.) when changing runtime code.

> **Architecture note (2026-06-05):** Frontend now runs on **Cloudflare Workers** (Worker `cyber-databrew` for prod, `cyber-databrew-dev` for dev), no longer on Cloud Run. Backend / SDK / Dagster still on GCP. See [Frontend (Cloudflare Workers)](#frontend--cloudflare-workers) below. Legacy Cloud Run frontend section is kept at the bottom and marked **DEPRECATED** for historical reference only.

When you finish implementing a change targeting Backend / Frontend / SDK / Dagster, you MUST NOT run `git commit` or `git push` until the following sequence has completed and the user has explicitly approved:

0. **Apply migrations** — If the diff includes new files under `backend/migrations/*.sql`, apply them to dev BEFORE building/deploying: `bash scripts/apply-migration-dev.sh "$(pwd)/backend/migrations/NNN_name.sql"`. Verify the migration succeeded before proceeding.
1. **Build & deploy depend on what changed:**
   - **`Frontend/` or `_worker.js` or `wrangler.jsonc` or `docs-site/`** → Cloudflare Worker. Run `wrangler deploy --env dev` locally (covered in [Frontend (Cloudflare Workers)](#frontend--cloudflare-workers) below). NO docker, NO Cloud Run.
   - **`backend/`** → Cloud Run. Build LOCALLY with `docker build` — NOT Cloud Build (`gcloud builds submit`). The repo has no `.gcloudignore`, so Cloud Build uploads a large tarball and is slow; local `docker build` is preferred. Use `--platform=linux/amd64` on ARM Macs (Cloud Run is amd64). On Docker Desktop + BuildKit, also pass `--output=type=docker` to force a single-platform Docker manifest — otherwise BuildKit produces a multi-platform OCI index that Cloud Run rejects with "Container manifest type must support amd64/linux".
     - **Tag with git SHA** (immutable) **and** push `cloudrun-dev-latest` (mutable convenience). See [Image tags and revision record](#image-tags-and-revision-record) below — do **not** push only `:cloudrun-dev-latest` without a SHA tag.
     - Do NOT call `bash deploy/cloudrun/backend-dev.sh` without `USE_EXISTING_IMAGE=true USE_CLOUD_BUILD=false`, because the script's default path is Cloud Build.
2. **Push image** (backend only) to Artifact Registry with `docker push` for **both** the SHA tag and `cloudrun-dev-latest` (auth via `gcloud auth configure-docker us-central1-docker.pkg.dev`). _Frontend has no image — wrangler uploads assets directly._
3. **Deploy** — Backend: `USE_EXISTING_IMAGE=true USE_CLOUD_BUILD=false IMAGE=<…>:<sha> bash deploy/cloudrun/backend-dev.sh`. Frontend: `wrangler deploy --env dev`. Then **record revision / version** (see below).
4. **Verify on dev** — Follow [`deploy-verification.md`](deploy-verification.md):
   - **Diff 含 `Frontend/`**：部署 frontend dev → `https://cyber-databrew-dev.cyberorigin.ai/` → Agent **必须**用 **Chrome DevTools MCP** 验收（截图 + console），不得默认让用户点浏览器。
   - **仅 backend / sdk 等（无 `Frontend/`）**：部署对应服务 + API smoke/curl；**不需要** Chrome DevTools MCP。
5. **Wait for user approval** — explicitly ask "确认部署 OK，可以 commit 吗？" (or equivalent). User must answer affirmatively.
6. **Pre-commit hook check (local)** — After user approves but BEFORE `git add`/`git commit`, run `pre-commit run --all-files` locally to catch trailing whitespace, YAML/JSON validation, secrets leakage, and other pre-commit issues. If any hook fails (excluding infra-only hooks like `tflint` that require tools not installed locally), fix the issue immediately. **Do not push commits that would fail CI pre-commit checks.**

Only after step 5 may you run `git add` / `git commit` / `git push`.

## Exceptions (no deploy required)

You MAY commit + push without the deploy gate when the change is **scope-isolated to non-runtime artifacts**:

- Pure documentation under `docs/`, `*.md`, `openspec/` (spec-only, no runtime code)
- CI / Tekton / GitHub Actions YAML that does not affect already-running services
- `.gcloudignore` / `.dockerignore` / `.gitignore`
- Test-only files that don't change product behavior

When in doubt, treat the change as runtime-affecting and follow the full gate.

## Image tags and revision record

**Why:** `:cloudrun-dev-latest` is overwritten on every deploy. Without a **git SHA image tag** and a **Cloud Run revision name**, you cannot roll back to the previous build.

**Defaults (do not change unless you know why):**

| Service | Hosting | Image / Worker name | Dev URL |
|---------|---------|---------------------|---------|
| Backend | Cloud Run | `us-central1-docker.pkg.dev/green-valley-442103/cyber-databrew-images/cyber-databrew-backend` → `cyber-databrew-backend-dev` | `https://api-cyber-databrew-dev.cyberorigin.ai/` |
| Frontend | **Cloudflare Worker** | Worker `cyber-databrew-dev` (env=production) | `https://cyber-databrew-dev.cyberorigin.ai/` |
| Frontend (prod) | **Cloudflare Worker** | Worker `cyber-databrew` (env=production) | `https://cyber-databrew.cyberorigin.ai/` |
| ~~Frontend (Cloud Run)~~ | ~~`cyber-databrew-frontend-dev`~~ | **DEPRECATED — migrated to Worker 2026-06-05.** Scripts under `deploy/cloudrun/frontend-*` retained for reference but no longer deployed. |

Set once per shell:

```bash
export PROJECT_ID=green-valley-442103
export REGION=us-central1
export SHA="$(git rev-parse --short HEAD)"
export REG=us-central1-docker.pkg.dev/${PROJECT_ID}
```

### Frontend — Cloudflare Workers

Frontend is a static SPA served by a **Cloudflare Worker** (`cyber-databrew-dev` for dev, `cyber-databrew` for prod). The Worker reverse-proxies `/api/*` to the Cloud Run backend (`-dev.` hostname routes to dev backend, otherwise prod — see `_worker.js`). **No docker, no Cloud Run, no nginx wrapper.**

#### Option A — Local wrangler (manual, today's default)

```bash
cd ~/cyber-databrew

# Build the SPA with the dev marker, refresh ./site/, rebuild /doc/, and deploy
INSTALL_DEPS=1 bash scripts/deploy-frontend-dev-worker.sh
# → cyber-databrew-dev Worker → cyber-databrew-dev.cyberorigin.ai

# Production deploy remains explicit:
# or
wrangler deploy               # → cyber-databrew (prod) Worker → cyber-databrew.cyberorigin.ai
```

`wrangler` reads `wrangler.jsonc` (top-level `name`, and `env.dev.name`/`env.dev.routes`). The custom domain is auto-bound by the `routes` block on first deploy — no dashboard step needed.

Required auth: `wrangler login` once (OAuth, stores creds outside the env). The `CLOUDFLARE_API_TOKEN` env var, if set, overrides OAuth — make sure it has the right scopes or `unset` it before deploy.

#### Option B — Cloudflare Workers Builds (auto on push)

Each Worker can be connected to this repo in the dashboard (Settings → Build → Connect Git). Build/deploy commands:

| Worker | Production branch | Build command | Deploy command |
|--------|-------------------|---------------|----------------|
| `cyber-databrew` (prod) | `dev` *(current)* or `main` *(recommended)* | `cd Frontend && npm ci && npm run build && mkdir -p ../site && cp -r dist/* ../site/` (+ docs-site optional) | `npx wrangler deploy` |
| `cyber-databrew-dev` | `dev` | `INSTALL_DEPS=1 DEPLOY_WORKER=0 bash scripts/deploy-frontend-dev-worker.sh` | `npx wrangler deploy --env dev` ⚠️ |

⚠️ The dev Worker deploy must use the canonical script or an equivalent command that sets `VITE_APP_ENV=dev`, refreshes `site/`, and deploys with `--env dev`.

Recommended Build watch paths to avoid spurious builds on backend-only changes:
```
Frontend/**
docs-site/**
_worker.js
wrangler.jsonc
site/**
```

#### Verify after deploy

```bash
# Frontend reachable
curl -sI https://cyber-databrew-dev.cyberorigin.ai/ | head -3
# expected: HTTP/2 200, server: cloudflare

# /api/* reverse-proxies to dev backend (not prod)
curl -sI https://cyber-databrew-dev.cyberorigin.ai/api/v1/<endpoint> | grep -i x-cloud-trace-context
# expected: a Cloud Run trace context header → confirms request landed on Cloud Run

# Prod untouched
curl -sI https://cyber-databrew.cyberorigin.ai/ | head -3
```

Record deploy evidence in the PR (see [Record revision](#record-revision-required-before-asking-to-commit)):

| Field | How to obtain |
|-------|----------------|
| Worker name | `cyber-databrew-dev` (or `cyber-databrew`) |
| Wrangler version ID | Tail of `wrangler deploy` output → `Current Version ID: <uuid>` |
| URL | `https://cyber-databrew-dev.cyberorigin.ai/` |
| Source SHA | `git rev-parse --short HEAD` at deploy time |

### Backend — build, push, deploy

```bash
export BACKEND_IMAGE="${REG}/cyber-databrew-images/cyber-databrew-backend:${SHA}"
export BACKEND_LATEST="${REG}/cyber-databrew-images/cyber-databrew-backend:cloudrun-dev-latest"

docker build --platform=linux/amd64 -f backend/Dockerfile -t "${BACKEND_IMAGE}" backend/
docker tag "${BACKEND_IMAGE}" "${BACKEND_LATEST}"
docker push "${BACKEND_IMAGE}"
docker push "${BACKEND_LATEST}"

USE_EXISTING_IMAGE=true USE_CLOUD_BUILD=false \
  IMAGE="${BACKEND_IMAGE}" \
  bash deploy/cloudrun/backend-dev.sh
```

`backend-dev.sh` carries the dev Cloud Run reachability defaults for DB, VPC,
Argo, and Kubernetes Pod diagnostics. Override only when targeting a different
dev environment:

| Setting | Default |
|---------|---------|
| `DB_HOST_OVERRIDE` | `172.27.160.7` |
| `DB_NAME_OVERRIDE` | `cyber_databrew_dev` |
| `DB_PASSWORD_SECRET` | `cyber-databrew-dev-postgres-password` |
| `VPC_CONNECTOR` | `cr-central-conn` |
| `ARGO_SERVER_URL_OVERRIDE` | `http://10.2.1.211:2746` |
| `K8S_API_ENDPOINT_OVERRIDE` | `https://34.59.48.233` |
| `K8S_BEARER_TOKEN_SECRET` | `cyber-databrew-dev-k8s-bearer-token` |
| `K8S_CA_DATA_SECRET` | `cyber-databrew-dev-k8s-ca-data` |

Do not hand-edit Cloud Run Console env vars for these keys. The next scripted
deploy will replace the revision template; keep the canonical values in this
script or pass explicit overrides.

### ~~Frontend (Cloud Run) — DEPRECATED~~

> ⚠️ **DEPRECATED 2026-06-05.** Frontend migrated to Cloudflare Workers (see [Frontend — Cloudflare Workers](#frontend--cloudflare-workers) above). The section below is kept for historical reference and rollback context only. **Do not use for new deploys.** Scripts under `deploy/cloudrun/frontend-*` are unmaintained.

Frontend is two images (SPA + nginx wrapper). Match `deploy/cloudrun/frontend-dev.sh` defaults:

```bash
export FRONTEND_BASE="${REG}/cyber-databrew-images/cyber-databrew-frontend:dev-latest"
export FRONTEND_IMAGE="${REG}/cyber-databrew-images/cyber-databrew-frontend:${SHA}"
export FRONTEND_LATEST="${REG}/cyber-databrew-images/cyber-databrew-frontend:cloudrun-dev-latest"

docker build --platform=linux/amd64 \
  --build-arg "VITE_APP_VERSION=${SHA}" \
  --build-arg "VITE_BUILD_REF=$(git rev-parse --abbrev-ref HEAD)#${SHA}" \
  -f Frontend/Dockerfile -t "${FRONTEND_BASE}" Frontend/

docker build --platform=linux/amd64 \
  --build-arg "BASE_IMAGE=${FRONTEND_BASE}" \
  -f deploy/cloudrun/frontend-cloudrun.Dockerfile -t "${FRONTEND_IMAGE}" .

docker tag "${FRONTEND_IMAGE}" "${FRONTEND_LATEST}"
docker push "${FRONTEND_BASE}"
docker push "${FRONTEND_IMAGE}"
docker push "${FRONTEND_LATEST}"

# Deploy only (skip rebuild — frontend-dev.sh always rebuilds)
gcloud run deploy cyber-databrew-frontend-dev \
  --quiet --project "${PROJECT_ID}" --region "${REGION}" --platform managed \
  --image "${FRONTEND_IMAGE}" --port 80 --allow-unauthenticated \
  --min-instances 0 --max-instances 5 --cpu 1 --memory 512Mi --timeout 60
```

**Alternatively**, run `bash deploy/cloudrun/frontend-dev.sh` with `IMAGE="${FRONTEND_IMAGE}"` and `BASE_IMAGE="${FRONTEND_BASE}"` — it will rebuild and push (slower, tags still land on `:${SHA}` if `IMAGE` is set).

### Record revision (required before asking to commit)

After each deploy, capture **revision name**, **service URL**, and **image tag** into PR and/or `openspec/changes/CYB-*/tasks.md`:

```bash
# Backend
gcloud run services describe cyber-databrew-backend-dev \
  --project "${PROJECT_ID}" --region "${REGION}" \
  --format='yaml(status.url,status.latestReadyRevisionName,spec.template.spec.containers[0].image)'

# Frontend
gcloud run services describe cyber-databrew-frontend-dev \
  --project "${PROJECT_ID}" --region "${REGION}" \
  --format='yaml(status.url,status.latestReadyRevisionName,spec.template.spec.containers[0].image)'
```

**Paste template (fill in):**

```markdown
### Deploy record — CYB-xxx
| Service | Image tag | Cloud Run revision | URL |
|---------|-----------|-------------------|-----|
| backend-dev | cyber-databrew-backend:`<sha>` | cyber-databrew-backend-dev-00xxx-abc | https://… |
| frontend-dev | cyber-databrew-frontend:`<sha>` | cyber-databrew-frontend-dev-00xxx-xyz | https://… |
```

**Tekton / dev deploy policy:**

- **Backend (Cloud Run):** Push to `dev` or feature branches **does not** auto-deploy (see `.tekton/push-backend-cloudrun-dev.yaml`, `on-cel-expression: false`). Use local `docker build` + `backend-dev.sh` (this doc), or open a PR and comment **`/deploy-cloudrun-dev`** on the PR. Prod `main` push pipelines unchanged. Tag images with git `SHA` in both paths.
- **Frontend (Cloudflare Workers):** Push to `dev` auto-deploys via **Cloudflare Workers Builds** if the Worker is connected to the repo in the dashboard (see [Option B](#option-b--cloudflare-workers-builds-auto-on-push) above). Otherwise deploy manually via `wrangler deploy --env dev`.

### Rollback (dev)

**Fastest — traffic only (previous revision still exists):**

```bash
gcloud run revisions list --service cyber-databrew-backend-dev \
  --project "${PROJECT_ID}" --region "${REGION}" \
  --format='table(metadata.name,metadata.creationTimestamp)'

gcloud run services update-traffic cyber-databrew-backend-dev \
  --project "${PROJECT_ID}" --region "${REGION}" \
  --to-revisions <PREVIOUS_REVISION_NAME>=100
```

**Redeploy old image** (when you recorded SHA tag):

```bash
USE_EXISTING_IMAGE=true USE_CLOUD_BUILD=false \
  IMAGE="${REG}/cyber-databrew-images/cyber-databrew-backend:<old-sha>" \
  DB_PASSWORD_SECRET=cyber-databrew-dev-postgres-password \
  bash deploy/cloudrun/backend-dev.sh
```

Full-stack rollback: revert **frontend Worker** (`wrangler rollback --env dev` or pick a previous Version ID in the dashboard) **and backend** to a matching deploy window (record both Worker version ID and Cloud Run revision in the PR).

## Rationale

Premature commits without a verified deploy create false confidence: the commit looks done, but the running revision is still stale. Commit should be the **last** action, not the first.

## Failure mode to avoid

❌ Make backend change → `go build` passes → `git commit` → `git push` → tell user "done"

✅ Make change → local tests → build image → push → deploy dev → targeted + regression verification (see `deploy-verification.md`) → PR evidence → ask user "verified, commit?" → wait → `git commit` → `git push`
