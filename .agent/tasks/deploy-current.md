# Task: Deploy Current Pipeline Changes to Cloud Run

## Background

Deploy the latest code (including uncommitted changes to `backend/internal/argo/client.go` and frontend fixes) to Cloud Run dev environment.

## Environment

- **Branch:** `feat/pipeline-integration`
- **Working directory:** `/Users/rick/cyber-databrew`
- **Project ID:** green-valley-442103
- **Region:** us-central1
- **Registry:** us-central1-docker.pkg.dev/green-valley-442103

## Backend Deploy Steps

### 1. Get SHA
```bash
SHA=$(git rev-parse --short HEAD)
```

### 2. Pre-pull BuildKit syntax image
```bash
docker pull docker/dockerfile:1 --platform=linux/amd64
```

### 3. Build backend image (ARM Mac → amd64)
```bash
REG="us-central1-docker.pkg.dev/green-valley-442103"
BACKEND_IMAGE="${REG}/rick-cyber-databrew-images/cyber-databrew-backend:${SHA}"
BACKEND_LATEST="${REG}/rick-cyber-databrew-images/cyber-databrew-backend:cloudrun-dev-latest"

docker build --platform=linux/amd64 --output=type=docker --provenance=false \
  -f backend/Dockerfile -t "${BACKEND_IMAGE}" backend/

docker tag "${BACKEND_IMAGE}" "${BACKEND_LATEST}"
```

### 4. Push to Artifact Registry
```bash
docker push "${BACKEND_IMAGE}"
docker push "${BACKEND_LATEST}"
```

### 5. Deploy using project script
```bash
USE_EXISTING_IMAGE=true USE_CLOUD_BUILD=false \
  IMAGE="${BACKEND_IMAGE}" \
  bash deploy/cloudrun/backend-dev.sh
```

### 6. Verify traffic is on the new revision
```bash
gcloud run services describe cyber-databrew-backend-dev \
  --project green-valley-442103 --region us-central1 \
  --format='yaml(status.latestReadyRevisionName,status.traffic)'
```

If traffic isn't on the new revision, migrate it:
```bash
gcloud run services update-traffic cyber-databrew-backend-dev \
  --to-revisions <NEW_REVISION_NAME>=100 \
  --project green-valley-442103 --region us-central1 --quiet
```

## Frontend Deploy Steps

### 1. Build frontend SPA
```bash
FRONTEND_BASE="${REG}/video-proc-images/cyber-databrew-frontend:dev-latest"
FRONTEND_IMAGE="${REG}/video-proc-images/cyber-databrew-frontend:${SHA}"
FRONTEND_LATEST="${REG}/video-proc-images/cyber-databrew-frontend:cloudrun-dev-latest"

docker build --platform=linux/amd64 --output=type=docker --provenance=false \
  --build-arg "VITE_APP_VERSION=${SHA}" \
  --build-arg "VITE_BUILD_REF=$(git rev-parse --abbrev-ref HEAD)#${SHA}" \
  -f Frontend/Dockerfile -t "${FRONTEND_BASE}" Frontend/
```

### 2. Build Cloud Run wrapper
```bash
docker build --platform=linux/amd64 --output=type=docker --provenance=false \
  --build-arg "BASE_IMAGE=${FRONTEND_BASE}" \
  -f deploy/cloudrun/frontend-cloudrun.Dockerfile -t "${FRONTEND_IMAGE}" .
```

### 3. Tag & push
```bash
docker tag "${FRONTEND_IMAGE}" "${FRONTEND_LATEST}"
docker push "${FRONTEND_BASE}"
docker push "${FRONTEND_IMAGE}"
docker push "${FRONTEND_LATEST}"
```

### 4. Deploy frontend
```bash
gcloud run deploy cyber-databrew-frontend-dev \
  --quiet --project green-valley-442103 --region us-central1 --platform managed \
  --image "${FRONTEND_IMAGE}" --port 80 --allow-unauthenticated \
  --min-instances 0 --max-instances 5 --cpu 1 --memory 512Mi --timeout 60
```

### 5. Verify
```bash
gcloud run services describe cyber-databrew-frontend-dev \
  --project green-valley-442103 --region us-central1 \
  --format='yaml(status.latestReadyRevisionName,status.traffic)'
```

## Important Notes
- `--output=type=docker --provenance=false` is REQUIRED on ARM Mac to avoid OCI index rejection by Cloud Run
- The deploy script (`backend-dev.sh`) needs `DB_PASSWORD_SECRET` env var — if it errors, fall back to direct `gcloud run deploy`
- If direct deploy fails due to DB_PASSWORD conflict (it's set via --set-secrets), use:
  ```bash
  gcloud run deploy cyber-databrew-backend-dev \
    --image "${BACKEND_IMAGE}" \
    --region us-central1 --platform managed --project green-valley-442103 --quiet
  ```
  This preserves existing env vars and secrets.

## Completion Criteria
- [ ] Backend Docker image pushed to GCR with SHA tag
- [ ] Backend deployed to Cloud Run, new revision serving traffic
- [ ] Frontend Docker image pushed to GCR with SHA tag
- [ ] Frontend deployed to Cloud Run, new revision serving traffic
- [ ] Verified with a quick API call to confirm backend is alive
