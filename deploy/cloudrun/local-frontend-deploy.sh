#!/usr/bin/env bash
# Local dev deploy: rsync Frontend/ → VM build → Cloud Run dev
# Usage: bash deploy/cloudrun/local-frontend-deploy.sh
set -euo pipefail

VM_HOST="136.119.82.209"
VM_USER="rick"
VM_KEY="${HOME}/.ssh/databrew-vm-key"
REPO_DIR="/home/rick/cyber-databrew"
BASE_IMAGE="us-central1-docker.pkg.dev/green-valley-442103/cyber-databrew-images/cyber-databrew-frontend:dev-latest"
IMAGE="us-central1-docker.pkg.dev/green-valley-442103/cyber-databrew-images/cyber-databrew-frontend:cloudrun-dev-latest"
BASE_CACHE_IMAGE="us-central1-docker.pkg.dev/green-valley-442103/cyber-databrew-images/cyber-databrew-frontend:base-buildcache"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
SSH="ssh -i ${VM_KEY} -o StrictHostKeyChecking=no -o ConnectTimeout=10"

echo "[1/3] syncing Frontend/ + deploy/cloudrun/ → VM (includes uncommitted changes)..."
rsync -az --delete \
  -e "ssh -i ${VM_KEY} -o StrictHostKeyChecking=no" \
  "${REPO_ROOT}/Frontend/" "${VM_USER}@${VM_HOST}:${REPO_DIR}/Frontend/"
rsync -az \
  -e "ssh -i ${VM_KEY} -o StrictHostKeyChecking=no" \
  "${SCRIPT_DIR}/frontend-cloudrun.Dockerfile" \
  "${SCRIPT_DIR}/frontend-nginx.conf" \
  "${VM_USER}@${VM_HOST}:${REPO_DIR}/deploy/cloudrun/"

echo "[2/3] building on VM..."
AR_TOKEN=$(gcloud auth print-access-token)
GIT_SHA=$(git -C "${REPO_ROOT}" rev-parse --short HEAD 2>/dev/null || echo "local")

${SSH} "${VM_USER}@${VM_HOST}" bash << ENDSSH
set -euo pipefail
echo "${AR_TOKEN}" | docker login -u oauth2accesstoken --password-stdin us-central1-docker.pkg.dev 2>&1 | grep -v "^$"

if docker buildx inspect databrew-frontend-builder &>/dev/null; then
  docker buildx use databrew-frontend-builder
  BASE_CACHE_ARGS="--cache-to=type=registry,ref=${BASE_CACHE_IMAGE},mode=max"
else
  docker buildx create --driver docker-container --use --name databrew-frontend-builder
  BASE_CACHE_ARGS="--cache-from=type=registry,ref=${BASE_CACHE_IMAGE} --cache-to=type=registry,ref=${BASE_CACHE_IMAGE},mode=max"
fi

echo "building SPA base image..."
docker buildx build \
  --progress=plain \
  --platform=linux/amd64 \
  \$BASE_CACHE_ARGS \
  --build-arg "VITE_APP_VERSION=${GIT_SHA}" \
  --build-arg "VITE_BUILD_REF=local#${GIT_SHA}" \
  -f ${REPO_DIR}/Frontend/Dockerfile \
  -t ${BASE_IMAGE} \
  --push \
  ${REPO_DIR}/Frontend

echo "building Cloud Run wrapper image..."
docker buildx build \
  --progress=plain \
  --platform=linux/amd64 \
  --build-arg "BASE_IMAGE=${BASE_IMAGE}" \
  -f ${REPO_DIR}/deploy/cloudrun/frontend-cloudrun.Dockerfile \
  -t ${IMAGE} \
  --push \
  ${REPO_DIR}

echo "pushed: ${IMAGE}"
ENDSSH

echo "[3/3] deploying to Cloud Run dev..."
IMAGE="${IMAGE}" \
BASE_IMAGE="${BASE_IMAGE}" \
USE_EXISTING_IMAGE=true \
BUILD_ARGO_UI=false \
bash "${SCRIPT_DIR}/frontend-dev.sh"

echo ""
echo "done. Cloud Run frontend-dev updated."
