#!/usr/bin/env bash
# init-index.sh — Create the `assets` index in Elasticsearch with proper mapping.
# Usage: ./init-index.sh [ELASTICSEARCH_URL]
#   Default ELASTICSEARCH_URL = http://localhost:9200
#
# Mapping design notes (see docs/review/data-platform-design.md §5.6.1):
#   - Time fields use `long` (nanosecond precision exceeds ES `date` ms range).
#   - `recorded_at` is a derived ms-precision date field for date_histogram aggs.
#   - `tags` is `nested` (keeps key/value/source/confidence association per tag),
#     plus `tags_flat` (flattened) for cheap equality filters in the common path.
#   - `algos` is `nested` to support multi-algo composite queries
#     ("hand_tracking score>0.8 AND face_blur=ok").
#   - `mcap.*` denormalizes high-cardinality device/scene fields from mcap_files
#     so search can filter by vendor/device/scene without joining PG.

set -euo pipefail

OS_URL="${1:-http://localhost:9200}"

# Optional HTTP Basic (e.g. when xpack.security.enabled=true). Set ELASTICSEARCH_PASSWORD
# (and optionally ELASTICSEARCH_USERNAME, default elastic).
_es_curl() {
  if [[ -n "${ELASTICSEARCH_PASSWORD:-}" ]]; then
    local u="${ELASTICSEARCH_USERNAME:-elastic}"
    curl -sf -u "${u}:${ELASTICSEARCH_PASSWORD}" "$@"
  else
    curl -sf "$@"
  fi
}

echo "⏳ Waiting for Elasticsearch at ${OS_URL} ..."
until _es_curl "${OS_URL}/_cluster/health" > /dev/null 2>&1; do
  sleep 2
done
echo "✅ Elasticsearch is ready."

# Delete existing index if present (idempotent re-run).
_es_curl -X DELETE "${OS_URL}/assets" > /dev/null 2>&1 || true

echo "📦 Creating 'assets' index with mapping ..."
_es_curl -X PUT "${OS_URL}/assets" \
  -H 'Content-Type: application/json' \
  -d '{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0
  },
  "mappings": {
    "properties": {
      "asset_id":           { "type": "keyword" },
      "mcap_file_id":       { "type": "keyword" },
      "segment_locator":    { "type": "keyword" },

      "asset_type":         { "type": "keyword" },
      "lifecycle_state":    { "type": "keyword" },
      "status":             { "type": "keyword" },
      "is_deleted":         { "type": "boolean" },
      "version":            { "type": "long" },
      "retention_tier":     { "type": "keyword" },
      "expire_at":          { "type": "date" },

      "storage_uri":        { "type": "keyword" },
      "thumb_uri":          { "type": "keyword" },
      "files":              { "type": "flattened" },

      "tenant_id":          { "type": "keyword" },
      "project_id":         { "type": "keyword" },

      "metadata":           { "type": "flattened" },

      "dataset": {
        "properties": {
          "format":             { "type": "keyword" },
          "record_count":       { "type": "long" },
          "size_bytes":         { "type": "long" },
          "annotation_status":  { "type": "keyword" }
        }
      },

      "annotation_result": {
        "properties": {
          "tool":           { "type": "keyword" },
          "quality_score":  { "type": "double" },
          "coverage":       { "type": "double" }
        }
      },

      "ml_model": {
        "properties": {
          "framework":    { "type": "keyword" },
          "architecture": { "type": "keyword" },
          "metrics":      { "type": "flattened" },
          "quantization": { "type": "keyword" },
          "artifact_uri": { "type": "keyword" }
        }
      },

      "evaluation_report": {
        "properties": {
          "model_id":   { "type": "keyword" },
          "dataset_id": { "type": "keyword" },
          "metrics":    { "type": "flattened" },
          "tool":       { "type": "keyword" }
        }
      },

      "owner":              { "type": "keyword", "fields": { "text": { "type": "text" } } },
      "reviewer":           { "type": "keyword", "fields": { "text": { "type": "text" } } },
      "notes":              { "type": "text" },

      "start_timestamp_ns": { "type": "long" },
      "end_timestamp_ns":   { "type": "long" },
      "duration_ms":        { "type": "long" },
      "recorded_at":        { "type": "date" },

      "created_at":         { "type": "date" },
      "updated_at":         { "type": "date" },

      "delivery_count":     { "type": "integer" },
      "last_delivered_at":  { "type": "date" },
      "last_delivered_to":  { "type": "keyword" },

      "parent_asset_id":    { "type": "keyword" },
      "root_asset_id":      { "type": "keyword" },
      "asset_level":        { "type": "integer" },

      "lineage_upstream_ids":   { "type": "keyword" },
      "lineage_downstream_ids": { "type": "keyword" },
      "lineage_relation_types": { "type": "keyword" },

      "logical_asset_id":   { "type": "keyword" },
      "revision":           { "type": "long" },
      "is_current":         { "type": "boolean" },

      "mcap": {
        "properties": {
          "mcap_uri":           { "type": "keyword" },
          "vendor_id":          { "type": "keyword" },
          "device_id":          { "type": "keyword" },
          "camera_model":       { "type": "keyword" },
          "scene_id":           { "type": "keyword" },
          "location_id":        { "type": "keyword" },
          "environment_id":     { "type": "keyword" },
          "task_id":            { "type": "keyword" },
          "data_source":        { "type": "keyword" },
          "file_duration_ms":   { "type": "long" },
          "recorded_at":        { "type": "date" }
        }
      },

      "tags_flat": { "type": "flattened" },

      "tags": {
        "type": "nested",
        "properties": {
          "key":         { "type": "keyword" },
          "value":       { "type": "keyword" },
          "value_num":   { "type": "double" },
          "value_bool":  { "type": "boolean" },
          "source_type": { "type": "keyword" },
          "source_name": { "type": "keyword" },
          "confidence":  { "type": "double" },
          "tagged_at":   { "type": "date" }
        }
      },

      "algos": {
        "type": "nested",
        "properties": {
          "name":         { "type": "keyword" },
          "version":      { "type": "keyword" },
          "status":       { "type": "keyword" },
          "result_tag":   { "type": "keyword" },
          "result_score": { "type": "double" },
          "run_id":       { "type": "keyword" },
          "finished_at":  { "type": "date" }
        }
      },

      "actions": {
        "type": "nested",
        "properties": {
          "action_id":    { "type": "keyword" },
          "start_ns":     { "type": "long" },
          "end_ns":       { "type": "long" },
          "labels":       { "type": "keyword" },
          "source_type":  { "type": "keyword" },
          "source_name":  { "type": "keyword" },
          "confidence":   { "type": "double" },
          "updated_at":   { "type": "date" }
        }
      }
    }
  }
}'

echo ""
echo "✅ Index 'assets' created successfully."
