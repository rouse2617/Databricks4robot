#!/usr/bin/env bash
# init-algo-runs-index.sh — Create the `algo_runs` index in Elasticsearch.
# Usage: ./init-algo-runs-index.sh [ELASTICSEARCH_URL]
#   Default ELASTICSEARCH_URL = http://localhost:9200

set -euo pipefail

OS_URL="${1:-http://localhost:9200}"

_es_curl() {
  if [[ -n "${ELASTICSEARCH_PASSWORD:-}" ]]; then
    local u="${ELASTICSEARCH_USERNAME:-elastic}"
    curl -sf -u "${u}:${ELASTICSEARCH_PASSWORD}" "$@"
  else
    curl -sf "$@"
  fi
}

echo "Waiting for Elasticsearch at ${OS_URL} ..."
until _es_curl "${OS_URL}/_cluster/health" > /dev/null 2>&1; do
  sleep 2
done
echo "Elasticsearch is ready."

_es_curl -X DELETE "${OS_URL}/algo_runs" > /dev/null 2>&1 || true

echo "Creating 'algo_runs' index with mapping ..."
_es_curl -X PUT "${OS_URL}/algo_runs" \
  -H 'Content-Type: application/json' \
  -d '{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0
  },
  "mappings": {
    "properties": {
      "run_id":            { "type": "keyword" },
      "algo_name":         { "type": "keyword" },
      "algo_version":      { "type": "keyword" },
      "algo_kind":         { "type": "keyword" },
      "triggered_by":      { "type": "keyword" },
      "status":            { "type": "keyword" },
      "row_version":       { "type": "long" },

      "tenant_id":         { "type": "keyword" },
      "project_id":        { "type": "keyword" },
      "pipeline_name":     { "type": "keyword" },
      "pipeline_version":  { "type": "keyword" },
      "code_commit":       { "type": "keyword" },
      "image_digest":      { "type": "keyword" },

      "started_at":        { "type": "date" },
      "finished_at":       { "type": "date" },
      "duration_ns":       { "type": "long" },
      "created_at":        { "type": "date" },
      "updated_at":        { "type": "date" },

      "assets_processed":  { "type": "integer" },
      "assets_succeeded":  { "type": "integer" },
      "assets_failed":     { "type": "integer" },
      "actions_created":   { "type": "integer" },
      "metrics_written":   { "type": "integer" },

      "cpu_seconds":       { "type": "long" },
      "gpu_seconds":       { "type": "long" },
      "cost_usd_micros":   { "type": "long" },

      "error_class":       { "type": "keyword" },
      "error_message":     { "type": "text" },

      "external_runtime":  { "type": "keyword" },
      "external_url":      { "type": "keyword" },

      "input_filter":      { "type": "flattened" },
      "input_asset_ids":   { "type": "keyword" },
      "params":            { "type": "flattened" },
      "outputs":           { "type": "flattened" }
    }
  }
}'

echo ""
echo "Index 'algo_runs' created successfully."
