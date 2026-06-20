#!/usr/bin/env bash
# Smoke: product Run API (dev Cloud Run).
#
# Usage:
#   bash scripts/smoke-runs-dev.sh
#   RUN_ID=<run-id> bash scripts/smoke-runs-dev.sh
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

echo "=== smoke-runs-dev === BASE=$BASE"

request GET "/api/v1/runs?view=summary&page=1&pageSize=1"
if [[ "$CODE" == "200" ]] && BODY_JSON="$BODY" python3 - <<'PY' >/dev/null 2>&1
import json, os
d = json.loads(os.environ["BODY_JSON"])
assert isinstance(d.get("items"), list)
assert "total" in d
PY
then
  ok "GET /runs summary"
  LIST_BODY="$BODY"
else
  bad "GET /runs summary"
  echo "=== done: ${PASS} passed, ${FAIL} failed ==="
  exit 1
fi

request GET "/api/v1/runs/watcher/status"
if [[ "$CODE" == "200" ]] && BODY_JSON="$BODY" python3 - <<'PY' >/dev/null 2>&1
import json, os
d = json.loads(os.environ["BODY_JSON"])
assert isinstance(d, dict)
PY
then
  ok "GET /runs/watcher/status"
else
  bad "GET /runs/watcher/status"
fi

RUN="${RUN_ID:-}"
if [[ -z "$RUN" ]]; then
	RUN=$(echo "$LIST_BODY" | python3 -c 'import sys,json; d=json.load(sys.stdin); items=d.get("items") or []; print(items[0]["id"] if items else "")')
fi

if [[ -z "$RUN" ]]; then
  echo "  SKIP no runs available; set RUN_ID to verify subresources"
else
  request GET "/api/v1/runs/${RUN}"
  if [[ "$CODE" == "200" ]]; then
    ok "GET /api/v1/runs/${RUN}"
    RUN_BODY="$BODY"
  else
    bad "GET /api/v1/runs/${RUN}"
    RUN_BODY=""
  fi

  WORKFLOW_NAME=$(BODY_JSON="$RUN_BODY" python3 - <<'PY'
import json, os
try:
    print(json.loads(os.environ["BODY_JSON"]).get("workflowName") or "")
except Exception:
    print("")
PY
)
  if [[ -n "$WORKFLOW_NAME" ]]; then
    request GET "/api/v1/runs/by-workflow/${WORKFLOW_NAME}"
    if [[ "$CODE" == "200" ]] && BODY_JSON="$BODY" RUN_ID="$RUN" python3 - <<'PY' >/dev/null 2>&1
import json, os
d = json.loads(os.environ["BODY_JSON"])
assert d.get("id") == os.environ["RUN_ID"]
PY
    then
      ok "GET /api/v1/runs/by-workflow/${WORKFLOW_NAME}"
    else
      bad "GET /api/v1/runs/by-workflow/${WORKFLOW_NAME}"
    fi
  fi

  for path in \
    "/api/v1/runs/${RUN}/events?limit=20" \
    "/api/v1/runs/${RUN}/nodes" \
    "/api/v1/runs/${RUN}/inputs" \
    "/api/v1/runs/${RUN}/outputs" \
    "/api/v1/runs/${RUN}/children" \
    "/api/v1/runs/${RUN}/runtime" \
    "/api/v1/runs/${RUN}/asset-nodes?limit=20" \
    "/api/v1/runs/${RUN}/cost-summary"; do
    request GET "$path"
    if [[ "$CODE" == "200" ]] && [[ "$path" == "/api/v1/runs/${RUN}/children" ]]; then
      if BODY_JSON="$BODY" python3 - <<'PY' >/dev/null 2>&1
import json, os
d = json.loads(os.environ["BODY_JSON"])
assert isinstance(d.get("items"), list)
assert isinstance(d.get("relations"), list)
summary = d.get("summary")
assert isinstance(summary, dict)
assert isinstance(summary.get("statuses"), dict)
assert isinstance(summary.get("aggregateStatus"), str)
assert summary.get("total") == d.get("total")
PY
      then
        ok "GET ${path} schema"
      else
        bad "GET ${path} schema"
      fi
    elif [[ "$CODE" == "200" ]]; then
      ok "GET ${path}"
    else
      bad "GET ${path}"
    fi
  done
fi

request GET "/api/v1/runs/not-a-real-run"
[[ "$CODE" == "404" ]] && ok "unknown run -> 404" || bad "unknown run"

request GET "/api/v1/runs/not-a-real-run/events"
[[ "$CODE" == "404" ]] && ok "unknown run events -> 404" || bad "unknown run events"

request GET "/api/v1/runs/not-a-real-run/inputs"
[[ "$CODE" == "404" ]] && ok "unknown run inputs -> 404" || bad "unknown run inputs"

request DELETE "/api/v1/runs/not-a-real-run"
[[ "$CODE" == "404" ]] && ok "unknown run delete -> 404" || bad "unknown run delete"

echo "=== done: ${PASS} passed, ${FAIL} failed ==="
[[ "$FAIL" -eq 0 ]]
