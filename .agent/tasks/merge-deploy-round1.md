# Task: Merge Round 1 Worktrees → Build → Deploy → Verify

## Background

Round 1 produced 4 parallel worktrees. Need to merge all into `feat/pipeline-integration`, build, deploy, and verify.

## Worktrees (branch → directory)

| Branch | Path | Content | Commit Status |
|--------|------|---------|---------------|
| `round1/fix-backend` | `/tmp/worktree-fix-backend` | Backend bug fixes (nil panic, CSRF, migration) | ✅ committed `9b2abf6` |
| `round1/fix-frontend` | `/tmp/worktree-fix-frontend` | Frontend bug fixes | ✅ committed `9c7b45d` |
| `round1/phase0-backend` | `/tmp/worktree-phase0-backend` | New Argo API handlers + routes + response fields | ⚠️ NOT committed (uncommitted changes) |
| `round1/phase-a-nodep` | `/tmp/worktree-phase-a-nodep` | Summary counts + name search | ✅ committed `d3cc529` |

## Steps

### 1. Merge committed worktrees
```bash
cd /Users/rick/cyber-databrew
git merge round1/fix-backend --no-edit
git merge round1/fix-frontend --no-edit
git merge round1/phase-a-nodep --no-edit
```

Handle any TASK.md conflicts: `rm TASK.md && git add TASK.md && git commit --no-edit`

### 2. Handle phase0-backend (uncommitted)
The phase0-backend worktree has uncommitted changes (Codex didn't commit due to deploy-before-commit gate).

Option A: Copy changed files manually (check git diff in the worktree)
```bash
cd /tmp/worktree-phase0-backend
# List changed files
git diff --name-only
```

Then either copy files or cherry-pick the diff.

### 3. Build & verify
```bash
cd /Users/rick/cyber-databrew
go build ./cmd/server
go test ./internal/argo/ ./internal/handlers/workflow/ ./routes/
cd Frontend && npm run lint
```

### 4. Deploy to Cloud Run

Use the standard deploy process:
```bash
cd /Users/rick/cyber-databrew

# Get SHA
SHA=$(git rev-parse --short HEAD)
REG="us-central1-docker.pkg.dev/green-valley-442103"

# Pre-pull syntax image
docker pull docker/dockerfile:1 --platform=linux/amd64

# Build backend (ARM Mac → amd64)
BACKEND_IMAGE="${REG}/rick-cyber-databrew-images/cyber-databrew-backend:${SHA}"
BACKEND_LATEST="${REG}/rick-cyber-databrew-images/cyber-databrew-backend:cloudrun-dev-latest"
docker build --platform=linux/amd64 --output=type=docker --provenance=false \
  -f backend/Dockerfile -t "${BACKEND_IMAGE}" backend/
docker tag "${BACKEND_IMAGE}" "${BACKEND_LATEST}"

# Push
docker push "${BACKEND_IMAGE}"
docker push "${BACKEND_LATEST}"

# Deploy
USE_EXISTING_IMAGE=true USE_CLOUD_BUILD=false \
  IMAGE="${BACKEND_IMAGE}" \
  bash deploy/cloudrun/backend-dev.sh

# Build frontend
FRONTEND_BASE="${REG}/video-proc-images/cyber-databrew-frontend:dev-latest"
FRONTEND_IMAGE="${REG}/video-proc-images/cyber-databrew-frontend:${SHA}"
FRONTEND_LATEST="${REG}/video-proc-images/cyber-databrew-frontend:cloudrun-dev-latest"

docker build --platform=linux/amd64 --output=type=docker --provenance=false \
  --build-arg "VITE_APP_VERSION=${SHA}" \
  --build-arg "VITE_BUILD_REF=$(git rev-parse --abbrev-ref HEAD)#${SHA}" \
  -f Frontend/Dockerfile -t "${FRONTEND_BASE}" Frontend/

docker build --platform=linux/amd64 --output=type=docker --provenance=false \
  --build-arg "BASE_IMAGE=${FRONTEND_BASE}" \
  -f deploy/cloudrun/frontend-cloudrun.Dockerfile -t "${FRONTEND_IMAGE}" .

docker tag "${FRONTEND_IMAGE}" "${FRONTEND_LATEST}"
docker push "${FRONTEND_BASE}"
docker push "${FRONTEND_IMAGE}"
docker push "${FRONTEND_LATEST}"

# Deploy frontend
gcloud run deploy cyber-databrew-frontend-dev \
  --quiet --project green-valley-442103 --region us-central1 --platform managed \
  --image "${FRONTEND_IMAGE}" --port 80 --allow-unauthenticated \
  --min-instances 0 --max-instances 5 --cpu 1 --memory 512Mi --timeout 60
```

### 5. Verify
```bash
# Backend health
curl -s -o /dev/null -w "%{http_code}" https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app/healthz

# Frontend
curl -s -o /dev/null -w "%{http_code}" https://cyber-databrew-frontend-dev-234851712830.us-central1.run.app/
```

### 6. Commit phase0-backend (after deploy)
After deploy verification, commit the phase0-backend changes:
```bash
cd /Users/rick/cyber-databrew
git add -A && git commit -m "feat(argo): phase 0 backend - add workflow operations api"
```

## Notes
- If backend-dev.sh errors on DB_PASSWORD, fall back to direct gcloud run deploy
- Check traffic: `gcloud run services describe cyber-databrew-backend-dev --region us-central1 --format='yaml(status.traffic)'`
