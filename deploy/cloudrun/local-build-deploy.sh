#!/usr/bin/env bash
# Local dev deploy: rsync current backend/ → VM build → Cloud Run dev
# Usage: bash deploy/cloudrun/local-build-deploy.sh
set -euo pipefail

VM_HOST="136.119.82.209"
VM_USER="rick"
VM_KEY="${HOME}/.ssh/databrew-vm-key"
REPO_DIR="/home/rick/cyber-databrew"
IMAGE="us-central1-docker.pkg.dev/green-valley-442103/cyber-databrew-images/cyber-databrew-backend:cloudrun-dev-latest"
CACHE_IMAGE="us-central1-docker.pkg.dev/green-valley-442103/cyber-databrew-images/cyber-databrew-backend:buildcache"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
SSH="ssh -i ${VM_KEY} -o StrictHostKeyChecking=no -o ConnectTimeout=10"

echo "[1/3] syncing backend/ → VM (includes uncommitted changes)..."
rsync -az --delete \
  -e "ssh -i ${VM_KEY} -o StrictHostKeyChecking=no" \
  "${REPO_ROOT}/backend/" "${VM_USER}@${VM_HOST}:${REPO_DIR}/backend/"

echo "[2/3] building on VM..."
AR_TOKEN=$(gcloud auth print-access-token)

${SSH} "${VM_USER}@${VM_HOST}" bash << ENDSSH
set -euo pipefail
echo "$AR_TOKEN" | docker login -u oauth2accesstoken --password-stdin us-central1-docker.pkg.dev 2>&1 | grep -v "^$"

if docker buildx inspect databrew-builder &>/dev/null; then
  docker buildx use databrew-builder
  CACHE_ARGS="--cache-to=type=registry,ref=${CACHE_IMAGE},mode=max"
else
  docker buildx create --driver docker-container --use --name databrew-builder
  CACHE_ARGS="--cache-from=type=registry,ref=${CACHE_IMAGE} --cache-to=type=registry,ref=${CACHE_IMAGE},mode=max"
fi

docker buildx build \
  --progress=plain \
  --platform=linux/amd64 \
  \$CACHE_ARGS \
  -f ${REPO_DIR}/backend/Dockerfile \
  -t ${IMAGE} \
  --push \
  ${REPO_DIR}/backend
echo "pushed: ${IMAGE}"
ENDSSH

echo "[3/3] deploying to Cloud Run dev..."
# ENABLE_GMP_SIDECAR=true keeps parity with deploy-dev.yml (CYB-4146): a local
# deploy must NOT silently strip the collector sidecar back to single-container.
IMAGE="${IMAGE}" \
USE_EXISTING_IMAGE=true \
SOURCE_K8S_ENV=false \
ENABLE_GMP_SIDECAR=true \
bash "${SCRIPT_DIR}/backend-dev.sh"

echo ""
echo "done. Cloud Run dev updated."
