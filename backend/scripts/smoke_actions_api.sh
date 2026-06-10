#!/usr/bin/env bash
# Smoke: health + optional actions list on a segment asset (requires stack + token).
# Usage:
#   DATABREW_TOKEN=dev-token bash backend/scripts/smoke_actions_api.sh
#   SEG_ASSET_ID=uuid DATABREW_TOKEN=dev-token bash backend/scripts/smoke_actions_api.sh
set -euo pipefail

BASE="${BASE_URL:-http://127.0.0.1:8080}"
TOKEN="${DATABREW_TOKEN:-dev-token}"

curl -sfS "$BASE/healthz" | grep -q '"status":"ok"' || {
  echo "smoke: healthz failed"
  exit 1
}
echo "smoke: healthz ok"

if [ -n "${SEG_ASSET_ID:-}" ]; then
  code="$(curl -sS -o /tmp/smoke_actions.json -w '%{http_code}' \
    -H "X-Databrew-Token: $TOKEN" \
    "$BASE/api/v1/assets/$SEG_ASSET_ID/actions")"
  if [ "$code" != "200" ]; then
    echo "smoke: GET actions HTTP $code"
    cat /tmp/smoke_actions.json
    exit 1
  fi
  echo "smoke: GET /assets/$SEG_ASSET_ID/actions ok ($(grep -o '"total":[0-9]*' /tmp/smoke_actions.json || echo total=?))"
else
  echo "smoke: skip actions list (set SEG_ASSET_ID to test)"
fi

echo "smoke: done"
