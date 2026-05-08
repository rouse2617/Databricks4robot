#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

BASE_URL="${BASE_URL:-http://localhost:8080}"
TOKEN="${TOKEN:-dev-token}"
BATCH_MARKER="${BATCH_MARKER:-cdc_loadtest_$(date +%Y%m%d%H%M%S)}"
COUNT="${COUNT:-5000}"
CONCURRENCY="${CONCURRENCY:-32}"

echo "Running CDC loadtest with:"
echo "  BASE_URL=${BASE_URL}"
echo "  BATCH_MARKER=${BATCH_MARKER}"
echo "  COUNT=${COUNT}"
echo "  CONCURRENCY=${CONCURRENCY}"

cd "${ROOT_DIR}/sdk"
uv run python ../backend/scripts/api_loadtest_assets.py \
  --base-url "${BASE_URL}" \
  --token "${TOKEN}" \
  --batch-marker "${BATCH_MARKER}" \
  --count "${COUNT}" \
  --concurrency "${CONCURRENCY}"

echo
echo "Loadtest finished. Next step:"
echo "  BATCH_MARKER=${BATCH_MARKER} make cdc-reconcile-report"
