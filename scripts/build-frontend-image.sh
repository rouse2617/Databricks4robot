#!/usr/bin/env bash
# Build Frontend image (Frontend/Dockerfile) for local or registry tags.
#
# Usage:
#   ./scripts/build-frontend-image.sh
#   IMAGE_TAG=us-central1-docker.pkg.dev/green-valley-442103/video-proc-images/cyber-databrew-frontend:dev-latest ./scripts/build-frontend-image.sh
#
# Optional:
#   VITE_API_BASE_URL=/api    # compile-time var for Vite build
#   PLATFORM=auto|native|linux/amd64|linux/arm64
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE_TAG="${IMAGE_TAG:-cyber-databrew-frontend:local}"
VITE_API_BASE_URL="${VITE_API_BASE_URL:-/api}"
VITE_DEV_ACCESS_TOKEN="${VITE_DEV_ACCESS_TOKEN:-}"
PLATFORM="${PLATFORM:-auto}"
# Sidebar build label: package version + git SHA (override with VITE_APP_VERSION / VITE_BUILD_REF)
DEFAULT_APP_VERSION="$(node -p "require('${ROOT}/Frontend/package.json').version" 2>/dev/null || echo 0.0.0)"
VITE_APP_VERSION="${VITE_APP_VERSION:-$DEFAULT_APP_VERSION}"
DEFAULT_BUILD_REF="$(git -C "${ROOT}" rev-parse --short HEAD 2>/dev/null || echo local)"
VITE_BUILD_REF="${VITE_BUILD_REF:-$DEFAULT_BUILD_REF}"

case "${PLATFORM}" in
auto)
  case "$(uname -m)" in
    arm64|aarch64) RESOLVED_PLATFORM=linux/amd64 ;;
    *) RESOLVED_PLATFORM="" ;;
  esac
  ;;
native|local) RESOLVED_PLATFORM="" ;;
*) RESOLVED_PLATFORM="${PLATFORM}" ;;
esac

args=(
	-t "${IMAGE_TAG}"
	-f "${ROOT}/Frontend/Dockerfile"
	--build-arg "VITE_API_BASE_URL=${VITE_API_BASE_URL}"
	--build-arg "VITE_APP_VERSION=${VITE_APP_VERSION}"
	--build-arg "VITE_BUILD_REF=${VITE_BUILD_REF}"
)
if [[ -n "${VITE_DEV_ACCESS_TOKEN}" ]]; then
  args+=(--build-arg "VITE_DEV_ACCESS_TOKEN=${VITE_DEV_ACCESS_TOKEN}")
fi
if [[ -n "${RESOLVED_PLATFORM}" ]]; then
  args=(--platform "${RESOLVED_PLATFORM}" "${args[@]}")
fi

docker build "${args[@]}" "${ROOT}/Frontend"
echo "Built ${IMAGE_TAG}"
