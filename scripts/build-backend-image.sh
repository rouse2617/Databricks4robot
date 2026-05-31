#!/usr/bin/env bash
# Same Docker context as Tekton buildkit (.tekton/*.yaml): CONTEXT=backend, Dockerfile=Dockerfile.
#
# Usage (repo root):
#   ./scripts/build-backend-image.sh
#   IMAGE_TAG=my-registry/cyber-databrew-backend:dev ./scripts/build-backend-image.sh
#
# Platform (--platform):
#   Default (PLATFORM unset or PLATFORM=auto): on ARM Mac / ARM Linux (uname arm64|aarch64),
#   use linux/amd64 so the image matches typical GKE amd64 nodes; otherwise no --platform.
#   PLATFORM=native|local — skip --platform (fast linux/arm64 image on Apple Silicon).
#   PLATFORM=linux/arm64 — force explicitly.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE_TAG="${IMAGE_TAG:-cyber-databrew-backend:local}"
PLATFORM="${PLATFORM:-auto}"

# Auto-detect version info from git (if available)
VERSION="$(cd "${ROOT}" && git describe --tags 2>/dev/null || echo "dev")"
COMMIT="$(cd "${ROOT}" && git rev-parse --short HEAD 2>/dev/null || echo "unknown")"

case "${PLATFORM}" in
auto)
	case "$(uname -m)" in
	arm64 | aarch64)
		RESOLVED_PLATFORM=linux/amd64
		;;
	*)
		RESOLVED_PLATFORM=""
		;;
	esac
	;;
native | local)
	RESOLVED_PLATFORM=""
	;;
*)
	RESOLVED_PLATFORM="${PLATFORM}"
	;;
esac

args=(-t "${IMAGE_TAG}" -f "${ROOT}/backend/Dockerfile"
      --build-arg "VERSION=${VERSION}"
      --build-arg "COMMIT=${COMMIT}")
if [[ -n "${RESOLVED_PLATFORM}" ]]; then
    args=(--platform "${RESOLVED_PLATFORM}" "${args[@]}")
fi

echo "Building backend image: ${IMAGE_TAG}"
echo "  VERSION=${VERSION}  COMMIT=${COMMIT}"

docker build "${args[@]}" "${ROOT}/backend"

echo "Built ${IMAGE_TAG}"
