#!/usr/bin/env bash
# Adds lineage projection fields to the existing assets index, then optionally
# triggers the backend admin reindex endpoint to rebuild document values.
#
# Usage:
#   deploy/local/elasticsearch/update-assets-lineage-mapping.sh [ELASTICSEARCH_URL]
#
# Optional env:
#   BACKEND_URL=http://localhost:8080
#   DATABREW_TOKEN=...
#   REINDEX=true

set -euo pipefail

OS_URL="${1:-${ELASTICSEARCH_URL:-http://localhost:9200}}"

_es_curl() {
  if [[ -n "${ELASTICSEARCH_PASSWORD:-}" ]]; then
    local u="${ELASTICSEARCH_USERNAME:-elastic}"
    curl -sf -u "${u}:${ELASTICSEARCH_PASSWORD}" "$@"
  else
    curl -sf "$@"
  fi
}

echo "Updating assets mapping at ${OS_URL}"
_es_curl -X PUT "${OS_URL}/assets/_mapping" \
  -H 'Content-Type: application/json' \
  -d '{
    "properties": {
      "lineage_upstream_ids":   { "type": "keyword" },
      "lineage_downstream_ids": { "type": "keyword" },
      "lineage_relation_types": { "type": "keyword" }
    }
  }'

if [[ "${REINDEX:-false}" == "true" ]]; then
  if [[ -z "${DATABREW_TOKEN:-}" ]]; then
    echo "DATABREW_TOKEN is required when REINDEX=true" >&2
    exit 1
  fi
  backend_url="${BACKEND_URL:-http://localhost:8080}"
  echo "Triggering backend reindex at ${backend_url}"
  curl -sf -X POST "${backend_url}/api/v1/admin/search/reindex" \
    -H "X-Databrew-Token: ${DATABREW_TOKEN}" \
    -H 'Content-Type: application/json' \
    -d '{"dry_run":false,"page_size":200}'
fi

echo "Done."
