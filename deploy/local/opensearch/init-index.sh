#!/usr/bin/env bash
# init-index.sh — Create the `assets` index in OpenSearch with proper mapping.
# Usage: ./init-index.sh [OPENSEARCH_URL]
#   Default OPENSEARCH_URL = http://localhost:9200

set -euo pipefail

OS_URL="${1:-http://localhost:9200}"

echo "⏳ Waiting for OpenSearch at ${OS_URL} ..."
until curl -sf "${OS_URL}/_cluster/health" > /dev/null 2>&1; do
  sleep 2
done
echo "✅ OpenSearch is ready."

# Delete existing index if present (idempotent re-run).
curl -sf -X DELETE "${OS_URL}/assets" > /dev/null 2>&1 || true

echo "📦 Creating 'assets' index with mapping ..."
curl -sf -X PUT "${OS_URL}/assets" \
  -H 'Content-Type: application/json' \
  -d '{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0
  },
  "mappings": {
    "properties": {
      "asset_id":        { "type": "keyword" },
      "mcap_file_id":    { "type": "keyword" },
      "segment_locator": { "type": "keyword" },
      "status":          { "type": "keyword" },
      "env":             { "type": "keyword" },
      "task":            { "type": "keyword" },
      "owner":           { "type": "keyword" },
      "reviewer":        { "type": "keyword" },
      "notes":           { "type": "text" },
      "batch":           { "type": "text" },
      "duration_sec":    { "type": "float" },
      "delivery_count":  { "type": "integer" },
      "tag_priority":    { "type": "keyword" },
      "tag_quality":     { "type": "keyword" },
      "tags":            { "type": "object", "dynamic": true },
      "algo_summary":    { "type": "object", "dynamic": true },
      "created_at":      { "type": "date" },
      "updated_at":      { "type": "date" }
    }
  }
}'

echo ""
echo "✅ Index 'assets' created successfully."
