#!/usr/bin/env bash
# Smoke: pipeline run events API (dev Cloud Run).
#
# Usage:
#   bash scripts/smoke-pipeline-run-events-dev.sh
#   RUN_ID=<pipeline-run-id> bash scripts/smoke-pipeline-run-events-dev.sh
#
# Requires: curl, python3; sources scripts/dev-backend-env.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=/dev/null
source "$ROOT/scripts/dev-backend-env.sh"

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
  local method="$1" path="$2"
  local raw
  raw=$(curl -sS --max-time 30 -w "\n%{http_code}" -X "$method" "${API_HDR[@]}" "${BASE}${path}" 2>/dev/null) || raw=$'\n000'
  CODE=$(echo "$raw" | tail -n1)
  BODY=$(echo "$raw" | sed '$d')
}

echo "=== smoke-pipeline-run-events-dev === BASE=$BASE"

RUN="${RUN_ID:-}"
if [[ -z "$RUN" ]]; then
  request GET "/api/v1/pipeline-runs"
  if [[ "$CODE" != "200" ]]; then
    bad "GET /pipeline-runs"
    echo "=== done: ${PASS} passed, ${FAIL} failed ==="
    exit 1
  fi
  RUN=$(echo "$BODY" | python3 -c 'import sys,json; d=json.load(sys.stdin); items=d.get("items") or []; print(items[0]["id"] if items else "")')
fi

if [[ -z "$RUN" ]]; then
  echo "  SKIP no pipeline runs available; set RUN_ID to verify a specific run"
else
  request GET "/api/v1/pipeline-runs/${RUN}/events?limit=20"
  if [[ "$CODE" == "200" ]] && echo "$BODY" | python3 -c 'import sys,json; d=json.load(sys.stdin); assert isinstance(d.get("items"), list); assert "total" in d' 2>/dev/null; then
    ok "GET /pipeline-runs/:id/events"
  else
    bad "GET /pipeline-runs/:id/events"
  fi

  request GET "/api/v1/pipeline-runs/${RUN}/events?limit=0"
  [[ "$CODE" == "400" ]] && ok "invalid limit -> 400" || bad "invalid limit"
fi

request GET "/api/v1/pipeline-runs/not-a-real-run/events"
[[ "$CODE" == "404" ]] && ok "unknown run -> 404" || bad "unknown run"

echo "=== done: ${PASS} passed, ${FAIL} failed ==="
[[ "$FAIL" -eq 0 ]]
