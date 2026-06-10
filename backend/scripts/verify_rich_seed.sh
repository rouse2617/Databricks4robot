#!/usr/bin/env bash
# Smoke-check after seed_rich_dataset.py: metrics search + asset list + eval registry reachability.
set -euo pipefail
BASE="${BASE:-http://127.0.0.1:8080}"
TOKEN="${TOKEN:-dev-token}"

echo "=== healthz ==="
curl -sS -o /dev/null -w "%{http_code}\n" -H "X-Databrew-Token: ${TOKEN}" "${BASE}/healthz"

echo "=== GET /api/v1/metrics/registry (count items) ==="
curl -sS -H "X-Databrew-Token: ${TOKEN}" "${BASE}/api/v1/metrics/registry" \
  | python3 -c "import sys,json; d=json.load(sys.stdin); print('items', len(d.get('items',[])))"

echo "=== POST /api/v1/metrics:search (good_frames_ratio >= 0, page 1, size 5) ==="
curl -sS -H "X-Databrew-Token: ${TOKEN}" -H "Content-Type: application/json" \
  -d '{"filters":{"lifecycle_state":"","metrics":[{"metric_key":"good_frames_ratio","op":"gte","value":0}]},"page":1,"page_size":5}' \
  "${BASE}/api/v1/metrics:search" | python3 -m json.tool

echo "=== POST /api/v1/queries/run (page 1, size 3) ==="
curl -sS -H "X-Databrew-Token: ${TOKEN}" -H "Content-Type: application/json" \
  -d '{"schema_version":"v1","mode":"structured","scope":{"resource":"assets"},"page":{"page":1,"page_size":3}}' \
  "${BASE}/api/v1/queries/run" \
  | python3 -c "import sys,json,re; d=json.load(sys.stdin); it=(d.get('items') or [{}])[0]; aid=it.get('asset_id',''); mid=it.get('mcap_file_id',''); ok=bool(mid and re.fullmatch(r'[0-9A-Za-z]{8}', mid)); print('total', d.get('total'), 'first_asset_id', aid, 'mcap_file_id', repr(mid), 'short_id_ok', ok)"

echo "OK"
