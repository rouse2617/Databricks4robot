#!/usr/bin/env bash
# Smoke test for the Argo run status push webhook (CYB-3058) against dev.
#
# Usage:
#   source scripts/dev-backend-env.sh          # exports BASE
#   export ARGO_RUN_WEBHOOK_TOKEN=<token>      # same token as the K8s Secret
#   bash scripts/smoke-argo-push-dev.sh [<known-workflow-name>]
#
# Covers: 401 (bad token), 400 (missing workflowName), 404 (unknown workflow),
# and — when a known workflow name is passed — 200 happy path.
set -euo pipefail

: "${BASE:?source scripts/dev-backend-env.sh first (BASE not set)}"
: "${ARGO_RUN_WEBHOOK_TOKEN:?set ARGO_RUN_WEBHOOK_TOKEN (must match the K8s Secret value)}"

URL="${BASE}/api/v1/pipeline-runs/webhook"
HDR_JSON=( -H "Content-Type: application/json" )
HDR_AUTH=( -H "X-Databrew-Webhook-Token: ${ARGO_RUN_WEBHOOK_TOKEN}" )

pass=0; fail=0
check() { # <label> <expected-code> <actual-code>
  if [[ "$2" == "$3" ]]; then echo "PASS $1 ($3)"; pass=$((pass+1));
  else echo "FAIL $1: expected $2 got $3"; fail=$((fail+1)); fi
}

# 1. bad token -> 401
code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$URL" \
  "${HDR_JSON[@]}" -H "X-Databrew-Webhook-Token: wrong-token" \
  -d '{"workflowName":"whatever"}')
check "bad-token->401" 401 "$code"

# 2. missing workflowName -> 400
code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$URL" \
  "${HDR_JSON[@]}" "${HDR_AUTH[@]}" -d '{}')
check "missing-workflowName->400" 400 "$code"

# 3. unknown workflow -> 404
code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$URL" \
  "${HDR_JSON[@]}" "${HDR_AUTH[@]}" \
  -d '{"workflowName":"does-not-exist-'"$RANDOM"'"}')
check "unknown-workflow->404" 404 "$code"

# 4. happy path (optional): pass a known dev workflow name as $1
if [[ -n "${1:-}" ]]; then
  resp=$(curl -s -X POST "$URL" "${HDR_JSON[@]}" "${HDR_AUTH[@]}" \
    -d '{"workflowName":"'"$1"'"}')
  echo "happy-path response: $resp"
  echo "$resp" | grep -q '"runId"' && { echo "PASS happy-path->200 (runId present)"; pass=$((pass+1)); } \
    || { echo "FAIL happy-path: no runId in $resp"; fail=$((fail+1)); }
fi

echo "---- smoke: $pass passed, $fail failed ----"
[[ "$fail" == "0" ]]
