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

args=(-t "${IMAGE_TAG}" -f "${ROOT}/backend/Dockerfile")
if [[ -n "${RESOLVED_PLATFORM}" ]]; then
	args=(--platform "${RESOLVED_PLATFORM}" "${args[@]}")
fi

docker build "${args[@]}" "${ROOT}/backend"

echo "Built ${IMAGE_TAG}"
