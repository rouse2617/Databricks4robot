#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

BATCH_MARKER="${BATCH_MARKER:-}"
export ELASTICSEARCH_URL="${ELASTICSEARCH_URL:-http://localhost:9200}"
export ES_INDEX="${ES_INDEX:-assets}"
export DB_HOST="${DB_HOST:-localhost}"
export DB_PORT="${DB_PORT:-5432}"
export DB_USER="${DB_USER:-postgres}"
export DB_PASSWORD="${DB_PASSWORD:-postgres}"
export DB_NAME="${DB_NAME:-cyber_databrew_dev}"

if [[ -n "${BATCH_MARKER}" ]]; then
  export CONSISTENCY_TARGET="${CONSISTENCY_TARGET:-0.999}"
  echo "Batch-scoped precheck for marker: ${BATCH_MARKER}"
  export PGPASSWORD="${DB_PASSWORD}"
  psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -c \
    "SELECT COUNT(*) AS pg_count FROM assets WHERE is_deleted = FALSE AND metadata->>'loadtest_batch'='${BATCH_MARKER}';"
  curl -s "${ELASTICSEARCH_URL}/${ES_INDEX}/_count" \
    -H 'Content-Type: application/json' \
    -d "{\"query\":{\"term\":{\"metadata.loadtest_batch.keyword\":\"${BATCH_MARKER}\"}}}" | jq .
  echo
fi

cd "${ROOT_DIR}"
bash backend/scripts/es-pg-audit.sh
