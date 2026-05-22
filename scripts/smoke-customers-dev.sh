#!/usr/bin/env bash
# Smoke: customers CRUD + delivery FK guard + deliveries filter (dev Cloud Run).
#
# Usage:
#   bash scripts/smoke-customers-dev.sh
#   ASSET_ID=dIN8Q1k2 bash scripts/smoke-customers-dev.sh
#
# Requires: curl, python3, gcloud; sources scripts/dev-backend-env.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=/dev/null
source "$ROOT/scripts/dev-backend-env.sh"

CID="${SMOKE_CUSTOMER_ID:-cyb-smoke-$(date +%s)}"
PASS=0
FAIL=0

ok() { PASS=$((PASS + 1)); echo "  OK  $1"; }
bad() {
  FAIL=$((FAIL + 1))
  echo "  FAIL $1 (HTTP ${CODE:-?})"
  echo "${BODY:-}" | head -c 400
  echo
}

request() {
  local method="$1" path="$2" data="${3:-}" extra_hdr="${4:-}"
  local raw curl_args=(-sS --max-time 30 -w "\n%{http_code}" -X "$method" "${API_HDR[@]}")
  [[ -n "$extra_hdr" ]] && curl_args+=(-H "$extra_hdr")
  if [[ -n "$data" ]]; then
    curl_args+=(-d "$data")
  fi
  raw=$(curl "${curl_args[@]}" "${BASE}${path}" 2>/dev/null) || raw=$'\n000'
  CODE=$(echo "$raw" | tail -n1)
  BODY=$(echo "$raw" | sed '$d')
}

echo "=== smoke-customers-dev === BASE=$BASE"

request POST "/api/v1/customers" "{\"customer_id\":\"${CID}\",\"display_name\":\"smoke customer\"}"
[[ "$CODE" == "201" ]] && ok "POST /customers" || bad "POST /customers"

request GET "/api/v1/customers/${CID}"
[[ "$CODE" == "200" ]] && ok "GET /customers/:id" || bad "GET /customers/:id"

ASSET="${ASSET_ID:-}"
if [[ -z "$ASSET" ]]; then
  echo "  SKIP delivery tests (set ASSET_ID=8-char asset id to enable)"
else
  IDEM_BAD="smoke-cust-bad-$(date +%s)"
  request POST "/api/v1/deliveries" "{\"customer_id\":\"zzz-not-a-customer\",\"asset_ids\":[\"${ASSET}\"]}" "Idempotency-Key: ${IDEM_BAD}"
  [[ "$CODE" == "422" ]] && ok "POST /deliveries unknown customer -> 422" || bad "POST /deliveries unknown customer"

  IDEM_OK="smoke-cust-ok-$(date +%s)"
  request POST "/api/v1/deliveries" "{\"customer_id\":\"${CID}\",\"asset_ids\":[\"${ASSET}\"]}" "Idempotency-Key: ${IDEM_OK}"
  [[ "$CODE" == "201" ]] && ok "POST /deliveries valid customer" || bad "POST /deliveries valid customer"

  request GET "/api/v1/deliveries?customer_id=${CID}&page=1&page_size=5"
  if [[ "$CODE" == "200" ]] && echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); assert d.get('total',0)>=1" 2>/dev/null; then
    ok "GET /deliveries?customer_id="
  else
    bad "GET /deliveries?customer_id="
  fi
fi

echo "=== done: ${PASS} passed, ${FAIL} failed ==="
[[ "$FAIL" -eq 0 ]]
