#!/usr/bin/env bash
# API smoke for local compose (Postgres + backend on :8080). Requires httpx/seed only for full runs.
# Usage:
#   GRACE_TOKEN=dev-token bash backend/scripts/api_smoke_compose.sh
#   BASE_URL=http://127.0.0.1:8080 FIRST_ASSET_ID=abcd1234 bash backend/scripts/api_smoke_compose.sh
set -euo pipefail

BASE="${BASE_URL:-http://127.0.0.1:8080}"
TOKEN="${GRACE_TOKEN:-dev-token}"
H=(-H "X-Grace-Token: ${TOKEN}" -H "Content-Type: application/json")

json_get() { curl -sS -fS "${H[@]}" "$1"; }

echo "=== GET /healthz ==="
json_get "${BASE}/healthz" | head -c 200
echo
echo

echo "=== GET /api/v1/metrics/registry (first keys) ==="
json_get "${BASE}/api/v1/metrics/registry" | python3 -c "import sys,json; d=json.load(sys.stdin); xs=d if isinstance(d,list) else d.get('items',d.get('metrics',[])); print('count', len(xs) if isinstance(xs,list) else 'n/a'); [print(' ',x.get('key',x)) for x in (xs[:5] if isinstance(xs,list) else [])]" 2>/dev/null || echo "(parse skip)"

echo "=== GET /api/v1/assets?page=1&page_size=2 ==="
ASSETS=$(curl -sS "${H[@]}" "${BASE}/api/v1/assets?page=1&page_size=2")
echo "$ASSETS" | python3 -m json.tool 2>/dev/null | head -n 40 || echo "$ASSETS" | head -c 600
echo

FIRST="${FIRST_ASSET_ID:-}"
if [ -z "$FIRST" ]; then
  FIRST=$(echo "$ASSETS" | python3 -c "import sys,json; d=json.load(sys.stdin); it=(d.get('items')or[{}])[0]; print(it.get('asset_id','') or '')" 2>/dev/null || true)
fi

if [ -n "$FIRST" ]; then
  echo
  echo "=== GET /api/v1/assets/${FIRST} (mcap + lifecycle) ==="
  curl -sS "${H[@]}" "${BASE}/api/v1/assets/${FIRST}" | python3 -c "import sys,json; d=json.load(sys.stdin); print('asset_id', d.get('asset_id')); print('mcap_file_id', d.get('mcap_file_id')); print('lifecycle_state', d.get('lifecycle_state'))"
  echo
  echo "=== GET /api/v1/assets/${FIRST}/tags ==="
  curl -sS "${H[@]}" "${BASE}/api/v1/assets/${FIRST}/tags" | python3 -m json.tool 2>/dev/null | head -n 25 || true
  echo
  echo "=== POST /api/v1/metrics:search (good_frames_ratio >= 0.05) ==="
  curl -sS "${H[@]}" -d '{"filters":{"lifecycle_state":"","metrics":[{"metric_key":"good_frames_ratio","op":"gte","value":0.05}]},"page":1,"page_size":5}' \
    "${BASE}/api/v1/metrics:search" | python3 -m json.tool 2>/dev/null | head -n 30 || true
else
  echo "(no asset id yet — run seed_rich_dataset.py)"
fi

echo
echo "=== GET /api/v1/mcap-files?page=1&page_size=2 ==="
curl -sS "${H[@]}" "${BASE}/api/v1/mcap-files?page=1&page_size=2" | python3 -m json.tool 2>/dev/null | head -n 35 || curl -sS "${H[@]}" "${BASE}/api/v1/mcap-files?page=1&page_size=2" | head -c 400

echo
echo "=== GET /api/v1/deliveries?page=1&page_size=3 ==="
curl -sS "${H[@]}" "${BASE}/api/v1/deliveries?page=1&page_size=3" | python3 -m json.tool 2>/dev/null | head -n 40 || true

echo
echo "=== GET /api/v1/customers/stress_customer/deliveries?page=1&page_size=3 ==="
curl -sS "${H[@]}" "${BASE}/api/v1/customers/stress_customer/deliveries?page=1&page_size=3" | python3 -m json.tool 2>/dev/null | head -n 30 || true

echo
echo "api_smoke_compose: done"
