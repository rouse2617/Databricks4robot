#!/usr/bin/env bash
# Smoke: delivery_rules + blocked POST /deliveries (dev Cloud Run).
#
# Usage:
#   bash scripts/smoke-delivery-rules-dev.sh
#   ASSET_ID=9KnuP7F3 CUSTOMER_ID=cyb-smoke-xxx bash scripts/smoke-delivery-rules-dev.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=/dev/null
source "$ROOT/scripts/dev-backend-env.sh"

ASSET="${ASSET_ID:-}"
CID="${CUSTOMER_ID:-cyb-drule-$(date +%s)}"
PASS=0
FAIL=0

ok() { PASS=$((PASS + 1)); echo "  OK  $1"; }
bad() {
  FAIL=$((FAIL + 1))
  echo "  FAIL $1 (HTTP ${CODE:-?})"
  echo "${BODY:-}" | head -c 500
  echo
}

request() {
  local method="$1" path="$2" data="${3:-}" extra_hdr="${4:-}"
  local raw curl_args=(-sS --max-time 30 -w "\n%{http_code}" -X "$method" "${API_HDR[@]}")
  [[ -n "$extra_hdr" ]] && curl_args+=(-H "$extra_hdr")
  [[ -n "$data" ]] && curl_args+=(-d "$data")
  raw=$(curl "${curl_args[@]}" "${BASE}${path}" 2>/dev/null) || raw=$'\n000'
  CODE=$(echo "$raw" | tail -n1)
  BODY=$(echo "$raw" | sed '$d')
}

echo "=== smoke-delivery-rules-dev === BASE=$BASE"

request POST "/api/v1/customers" "{\"customer_id\":\"${CID}\",\"display_name\":\"drule smoke\"}"
[[ "$CODE" == "201" ]] && ok "POST /customers" || bad "POST /customers"

request POST "/api/v1/delivery-rules" "{
  \"name\": \"no_pii\",
  \"owner\": \"smoke\",
  \"customer_id\": \"${CID}\",
  \"enforce_mode\": \"block\",
  \"query_dsl\": {\"where\": [{\"field\": \"tag.quality\", \"op\": \"eq\", \"value\": \"poor\"}]}
}"
[[ "$CODE" == "201" ]] && ok "POST /delivery-rules" || bad "POST /delivery-rules"

request GET "/api/v1/delivery-rules?customer_id=${CID}"
[[ "$CODE" == "200" ]] && ok "GET /delivery-rules" || bad "GET /delivery-rules"

if [[ -z "$ASSET" ]]; then
  echo "  SKIP delivery block test (set ASSET_ID=8-char asset)"
else
  request POST "/api/v1/assets/${ASSET}/tags" "{
    \"key\": \"quality\",
    \"value\": \"poor\",
    \"source_type\": \"human\",
    \"source_name\": \"smoke-drule\"
  }"
  if [[ "$CODE" == "200" || "$CODE" == "201" ]]; then
    ok "POST tag quality=poor on asset"
  else
    bad "POST tag on asset"
  fi

  IDEM="smoke-drule-block-$(date +%s)"
  request POST "/api/v1/deliveries" "{\"customer_id\":\"${CID}\",\"asset_ids\":[\"${ASSET}\"]}" "Idempotency-Key: ${IDEM}"
  if [[ "$CODE" == "422" ]] && echo "$BODY" | grep -q 'DELIVERY_RULE_FAILED'; then
    ok "POST /deliveries blocked by rule"
  else
    bad "POST /deliveries expected 422 DELIVERY_RULE_FAILED"
  fi
fi

echo "=== done: ${PASS} passed, ${FAIL} failed ==="
[[ "$FAIL" -eq 0 ]]
