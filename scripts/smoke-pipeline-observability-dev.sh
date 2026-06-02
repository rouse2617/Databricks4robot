#!/usr/bin/env bash
# Smoke: pipeline observability APIs (dev Cloud Run).
#
# Usage:
#   bash scripts/smoke-pipeline-observability-dev.sh
#   RUN_ID=<pipeline-run-id> bash scripts/smoke-pipeline-observability-dev.sh
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

echo "=== smoke-pipeline-observability-dev === BASE=$BASE"

RUN="${RUN_ID:-}"
if [[ -z "$RUN" ]]; then
  request GET "/api/v1/pipeline-runs"
  if [[ "$CODE" != "200" ]]; then
    bad "GET /pipeline-runs"
    exit 1
  fi
  RUN=$(echo "$BODY" | python3 -c 'import sys,json; d=json.load(sys.stdin); items=d.get("items") or []; print(items[0]["id"] if items else "")')
fi

if [[ -z "$RUN" ]]; then
  echo "  SKIP no pipeline runs available; set RUN_ID to verify a specific run"
else
  request GET "/api/v1/pipeline-runs/${RUN}/events?limit=20&q=node"
  if [[ "$CODE" == "200" ]] && echo "$BODY" | python3 -c 'import sys,json; d=json.load(sys.stdin); assert isinstance(d.get("items"), list)' 2>/dev/null; then
    ok "GET /pipeline-runs/:id/events?q="
  else
    bad "GET /pipeline-runs/:id/events?q="
  fi

  request GET "/api/v1/pipeline-runs/${RUN}/asset-nodes?limit=50&orderBy=cost"
  if [[ "$CODE" == "200" ]] && echo "$BODY" | python3 -c 'import sys,json; d=json.load(sys.stdin); assert isinstance(d.get("items"), list); assert "summary" in d' 2>/dev/null; then
    ok "GET /pipeline-runs/:id/asset-nodes"
  else
    bad "GET /pipeline-runs/:id/asset-nodes"
  fi

  request GET "/api/v1/pipeline-runs/${RUN}/cost-summary"
  if [[ "$CODE" == "200" ]] && echo "$BODY" | python3 -c 'import sys,json; d=json.load(sys.stdin); assert d.get("runId")' 2>/dev/null; then
    ok "GET /pipeline-runs/:id/cost-summary"
  else
    bad "GET /pipeline-runs/:id/cost-summary"
  fi
fi

request GET "/api/v1/pipeline-runs/not-a-real-run/asset-nodes"
[[ "$CODE" == "404" ]] && ok "unknown run asset-nodes -> 404" || bad "unknown run asset-nodes"

echo "=== done: ${PASS} passed, ${FAIL} failed ==="
[[ "$FAIL" -eq 0 ]]
