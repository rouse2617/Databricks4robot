#!/usr/bin/env bash
set -euo pipefail

PROJECT_ID="${PROJECT_ID:-green-valley-442103}"
REGION="${REGION:-us-central1}"
SERVICE_NAME="${SERVICE_NAME:-cyber-databrew-frontend-dev}"
BASE_IMAGE="${BASE_IMAGE:-us-central1-docker.pkg.dev/green-valley-442103/cyber-databrew-images/cyber-databrew-frontend:dev-latest}"
IMAGE="${IMAGE:-us-central1-docker.pkg.dev/green-valley-442103/cyber-databrew-images/cyber-databrew-frontend:cloudrun-dev-latest}"
USE_CLOUD_BUILD="${USE_CLOUD_BUILD:-false}"
BUILD_ARGO_UI="${BUILD_ARGO_UI:-auto}" # auto | true | false
ARGO_UI_DIST_SOURCE="${ARGO_UI_DIST_SOURCE:-}"

# ── Auto-detect version info from git ──
GIT_SHA="${GIT_SHA:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")}"
GIT_TAG="${GIT_TAG:-$(git describe --tags --always 2>/dev/null || echo "dev")}"
GIT_BRANCH="${GIT_BRANCH:-$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")}"
APP_VERSION="${APP_VERSION:-${GIT_TAG}}"
BUILD_REF="${BUILD_REF:-${GIT_BRANCH}#${GIT_SHA}}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
CLOUDBUILD_CFG="${SCRIPT_DIR}/frontend-cloudbuild.yaml"
FRONTEND_DOCKERFILE="${REPO_ROOT}/Frontend/Dockerfile"
FRONTEND_CLOUDRUN_DOCKERFILE="${SCRIPT_DIR}/frontend-cloudrun.Dockerfile"
ARGO_UI_DIR="${REPO_ROOT}/databrew-pipeline/argo-ui"
ARGO_UI_DIST="${ARGO_UI_DIR}/dist"
ARGO_UI_DIST_FALLBACKS=(
  "${REPO_ROOT}/site/argo"
)

copy_argo_ui_dist() {
  local source="$1"
  local reason="$2"
  if [[ ! -f "${source}/index.html" ]]; then
    echo "ERROR: ${reason} does not look like a built Argo UI dist: ${source}" >&2
    exit 1
  fi
  echo "Using Argo UI dist from ${reason}: ${source}"
  rm -rf "${ARGO_UI_DIST}"
  mkdir -p "$(dirname "${ARGO_UI_DIST}")"
  cp -R "${source}" "${ARGO_UI_DIST}"
}

ensure_argo_ui_dist() {
  if [[ "${BUILD_ARGO_UI}" == "false" ]]; then
    echo "Skipping Argo UI dist check (BUILD_ARGO_UI=false)."
    return 0
  fi

  if [[ -n "${ARGO_UI_DIST_SOURCE}" ]]; then
    copy_argo_ui_dist "${ARGO_UI_DIST_SOURCE}" "ARGO_UI_DIST_SOURCE"
    return 0
  fi

  if [[ -f "${ARGO_UI_DIST}/index.html" && "${BUILD_ARGO_UI}" != "true" ]]; then
    echo "Argo UI dist already exists: ${ARGO_UI_DIST}"
    return 0
  fi

  if [[ "${BUILD_ARGO_UI}" != "true" ]]; then
    for fallback in "${ARGO_UI_DIST_FALLBACKS[@]}"; do
      if [[ -f "${fallback}/index.html" ]]; then
        copy_argo_ui_dist "${fallback}" "repo fallback"
        return 0
      fi
    done
  fi

  if ! command -v yarn >/dev/null 2>&1; then
    echo "ERROR: databrew-pipeline/argo-ui/dist is missing and yarn is not installed." >&2
    echo "       Install yarn, add a valid site/argo bundle, set ARGO_UI_DIST_SOURCE=/path/to/dist, or set BUILD_ARGO_UI=false only if /argo is intentionally omitted." >&2
    exit 1
  fi

  echo "Building Argo UI dist for /argo route..."
  (
    cd "${ARGO_UI_DIR}"
    yarn install --frozen-lockfile
    yarn build
  )
}

ensure_argo_ui_dist

echo "Building SPA base image: ${BASE_IMAGE}"
echo "  Version: ${APP_VERSION}, Ref: ${BUILD_REF}"
docker build \
  --platform linux/amd64 \
  --build-arg "VITE_APP_VERSION=${APP_VERSION}" \
  --build-arg "VITE_BUILD_REF=${BUILD_REF}" \
  -f "${FRONTEND_DOCKERFILE}" \
  -t "${BASE_IMAGE}" \
  "${REPO_ROOT}/Frontend"

echo "Building Cloud Run frontend image: ${IMAGE}"
docker build \
  --build-arg "BASE_IMAGE=${BASE_IMAGE}" \
  --platform linux/amd64 \
  -f "${FRONTEND_CLOUDRUN_DOCKERFILE}" \
  -t "${IMAGE}" \
  "${REPO_ROOT}"

echo "Pushing images..."
docker push "${BASE_IMAGE}"
docker push "${IMAGE}"

echo "Deploying ${SERVICE_NAME} to Cloud Run (${REGION}, ${PROJECT_ID})"
gcloud run deploy "${SERVICE_NAME}" \
  --quiet \
  --project "${PROJECT_ID}" \
  --region "${REGION}" \
  --platform managed \
  --image "${IMAGE}" \
  --port 80 \
  --allow-unauthenticated \
  --min-instances 0 \
  --max-instances 5 \
  --cpu 1 \
  --memory 512Mi \
  --timeout 60

echo "Deployment complete."
gcloud run services describe "${SERVICE_NAME}" \
  --project "${PROJECT_ID}" \
  --region "${REGION}" \
  --platform managed \
  --format='value(status.url)'
