#!/usr/bin/env bash
# Build Frontend image (Frontend/Dockerfile) for local or registry tags.
#
# Usage:
#   ./scripts/build-frontend-image.sh
#   IMAGE_TAG=us-central1-docker.pkg.dev/<project>/<repo>/cyber-databrew-frontend:dev-latest ./scripts/build-frontend-image.sh
#
# Optional:
#   VITE_API_BASE_URL=/api    # compile-time var for Vite build
#   PLATFORM=auto|native|linux/amd64|linux/arm64
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE_TAG="${IMAGE_TAG:-cyber-databrew-frontend:local}"
VITE_API_BASE_URL="${VITE_API_BASE_URL:-/api}"
PLATFORM="${PLATFORM:-auto}"

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

args=(-t "${IMAGE_TAG}" -f "${ROOT}/Frontend/Dockerfile" --build-arg "VITE_API_BASE_URL=${VITE_API_BASE_URL}")
if [[ -n "${RESOLVED_PLATFORM}" ]]; then
  args=(--platform "${RESOLVED_PLATFORM}" "${args[@]}")
fi

docker build "${args[@]}" "${ROOT}/Frontend"
echo "Built ${IMAGE_TAG}"
