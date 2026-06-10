# Deploy-before-commit gate (REQUIRED)

Applies to **all AI agents** (Cursor, Codex, Claude Code, etc.) when changing runtime code.

When you finish implementing a change targeting Backend / Frontend / SDK / Dagster, you MUST NOT run `git commit` or `git push` until the following sequence has completed and the user has explicitly approved:

0. **Apply migrations** — If the diff includes new files under `backend/migrations/*.sql`, apply them to dev BEFORE building/deploying: `bash scripts/apply-migration-dev.sh "$(pwd)/backend/migrations/NNN_name.sql"`. Verify the migration succeeded before proceeding.
1. **Build image LOCALLY** with `docker build` — NOT Cloud Build (`gcloud builds submit`). The repo has no `.gcloudignore`, so Cloud Build uploads a large tarball and is slow; local `docker build` is preferred. Use `--platform=linux/amd64` on ARM Macs (Cloud Run is amd64). On Docker Desktop + BuildKit, also pass `--output=type=docker` to force a single-platform Docker manifest — otherwise BuildKit produces a multi-platform OCI index that Cloud Run rejects with "Container manifest type must support amd64/linux".
   - **Tag with git SHA** (immutable) **and** push `cloudrun-dev-latest` (mutable convenience). See [Image tags and revision record](#image-tags-and-revision-record) below — do **not** push only `:cloudrun-dev-latest` without a SHA tag.
   - Backend / Frontend build commands: same section below.
   - Do NOT call `bash deploy/cloudrun/backend-dev.sh` / `frontend-dev.sh` without `USE_EXISTING_IMAGE=true USE_CLOUD_BUILD=false`, because the script's default path is Cloud Build.
2. **Push image** to Artifact Registry with `docker push` for **both** the SHA tag and `cloudrun-dev-latest` (auth via `gcloud auth configure-docker us-central1-docker.pkg.dev`).
3. **Deploy** to Cloud Run dev using the **SHA-tagged** image: `USE_EXISTING_IMAGE=true USE_CLOUD_BUILD=false IMAGE=<…>:<sha> … bash deploy/cloudrun/backend-dev.sh` (or frontend). Then **record revision + image tag** (see below).
4. **Verify on dev** — Follow [`deploy-verification.md`](deploy-verification.md):
   - **Diff 含 `Frontend/`**：部署 frontend dev → Agent **必须**用 **Chrome DevTools MCP** 验收（截图 + console），不得默认让用户点浏览器。
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

| Service | Registry image | Dev service name |
|---------|----------------|------------------|
| Backend | `us-central1-docker.pkg.dev/green-valley-442103/cyber-databrew-images/cyber-databrew-backend` | `cyber-databrew-backend-dev` |
| Frontend (Cloud Run) | `us-central1-docker.pkg.dev/green-valley-442103/cyber-databrew-images/cyber-databrew-frontend` | `cyber-databrew-frontend-dev` |

Set once per shell:

```bash
export PROJECT_ID=green-valley-442103
export REGION=us-central1
export SHA="$(git rev-parse --short HEAD)"
export REG=us-central1-docker.pkg.dev/${PROJECT_ID}
```

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
  DB_PASSWORD_SECRET=cyber-databrew-dev-postgres-password \
  bash deploy/cloudrun/backend-dev.sh
```

### Frontend — build, push, deploy

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

**Tekton / dev deploy policy:** Push to `dev` or feature branches **does not** auto-deploy Cloud Run (see `.tekton/push-*-cloudrun-dev.yaml`, `on-cel-expression: false`). Use **local** build + `backend-dev.sh` / `frontend-dev.sh` (this doc), or open a PR to `main`/`dev` and comment **`/deploy-cloudrun-dev`** on the PR. Prod `main` push pipelines are unchanged. Tag images with git `SHA` in both paths.

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

Full-stack rollback: revert **both** frontend and backend to the **same** deploy window (matching SHA pair).

## Rationale

Premature commits without a verified deploy create false confidence: the commit looks done, but the running revision is still stale. Commit should be the **last** action, not the first.

## Failure mode to avoid

❌ Make backend change → `go build` passes → `git commit` → `git push` → tell user "done"

✅ Make change → local tests → build image → push → deploy dev → targeted + regression verification (see `deploy-verification.md`) → PR evidence → ask user "verified, commit?" → wait → `git commit` → `git push`
