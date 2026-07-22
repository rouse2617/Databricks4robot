#!/usr/bin/env bash
# Smoke test for the subscription-tasks CRUD API (CYB-3778) against dev.
#
# Usage:
#   source scripts/dev-backend-env.sh          # exports BASE + DATABREW_TOKEN
#   bash scripts/smoke-subscription-tasks-dev.sh
#
# Covers: create/get/pause/resume/delete happy path + 400 (missing required,
# empty bindings, bad interval) + 404 (unknown id). Leaves no rows on success.
set -euo pipefail

: "${BASE:?source scripts/dev-backend-env.sh first (BASE not set)}"
: "${DATABREW_TOKEN:?set DATABREW_TOKEN}"

URL="${BASE}/api/v1/subscription-tasks"
HDR=( -H "X-Databrew-Token: ${DATABREW_TOKEN}" -H "Content-Type: application/json" )

pass=0; fail=0
check() { # $1=name, $2=expected-code, $3=actual-code, $4=optional-body
    if [[ "$3" == "$2" ]]; then
        printf '  ✅ %s → %s\n' "$1" "$3"; pass=$((pass+1))
    else
        printf '  ❌ %s → %s (want %s)\n     body: %s\n' "$1" "$3" "$2" "${4:-}"; fail=$((fail+1))
    fi
}

# 400: missing required fields
code=$(curl -sS -o /tmp/smoke-st-400.txt -w '%{http_code}' -X POST "${HDR[@]}" "$URL" -d '{"name":"x"}')
check "create missing required is 400" 400 "$code" "$(cat /tmp/smoke-st-400.txt)"

# 400: no pipeline bindings
body='{"name":"bad","projectId":"p","subscriptionId":"s","pipelineBindings":[]}'
code=$(curl -sS -o /tmp/smoke-st-400b.txt -w '%{http_code}' -X POST "${HDR[@]}" "$URL" -d "$body")
check "create with empty bindings is 400" 400 "$code" "$(cat /tmp/smoke-st-400b.txt)"

# 400: bad pullIntervalSeconds
body='{"name":"bad","projectId":"p","subscriptionId":"s","pullIntervalSeconds":0,"pipelineBindings":[{"templateId":"t","targetId":"g"}]}'
code=$(curl -sS -o /tmp/smoke-st-400c.txt -w '%{http_code}' -X POST "${HDR[@]}" "$URL" -d "$body")
check "create with pullInterval=0 is 400" 400 "$code" "$(cat /tmp/smoke-st-400c.txt)"

# Happy path: create → get → pause → resume → delete.
NAME="smoke-subtask-$(date +%s)"
body=$(cat <<JSON
{
  "name": "$NAME",
  "enabled": false,
  "projectId": "green-valley-442103",
  "subscriptionId": "smoke-test-sub",
  "pullIntervalSeconds": 60,
  "maxMessagesPerPull": 100,
  "pipelineBindings": [
    {"templateId": "tpl-smoke-a", "targetId": "cluster-default"},
    {"templateId": "tpl-smoke-b", "targetId": "cluster-default", "templateVersion": 2}
  ]
}
JSON
)
resp=$(curl -sS -w '\nHTTPCODE:%{http_code}' -X POST "${HDR[@]}" "$URL" -d "$body")
code="${resp##*HTTPCODE:}"; json="${resp%HTTPCODE:*}"
check "create happy path is 201" 201 "$code" "$json"
ID=$(printf '%s' "$json" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("id",""))')
if [[ -z "$ID" ]]; then echo "  ❌ could not extract id from create body"; fail=$((fail+1)); ID="sub_unknown"; fi

code=$(curl -sS -o /dev/null -w '%{http_code}' "${HDR[@]}" "$URL/$ID")
check "get by id is 200" 200 "$code"

code=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "${HDR[@]}" "$URL/$ID/pause")
check "pause is 200" 200 "$code"

code=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "${HDR[@]}" "$URL/$ID/resume")
check "resume is 200" 200 "$code"

# CYB-3798: dispatch-history endpoints (read-only, reuse the Backfill surface).
# The smoke task is disabled and never dispatched, so its batch list is expected
# empty — we assert the endpoints answer 200, not that rows exist.
code=$(curl -sS -o /tmp/smoke-st-hist.txt -w '%{http_code}' "${HDR[@]}" "${BASE}/api/v1/backfill?createdBy=subscription-task:$ID")
check "list batches by createdBy is 200" 200 "$code" "$(cat /tmp/smoke-st-hist.txt)"

code=$(curl -sS -o /tmp/smoke-st-items.txt -w '%{http_code}' "${HDR[@]}" "${BASE}/api/v1/backfill/batch_definitely_not_here/items")
check "items of unknown batch is 200 (empty list)" 200 "$code" "$(cat /tmp/smoke-st-items.txt)"

# 404: unknown id
code=$(curl -sS -o /dev/null -w '%{http_code}' "${HDR[@]}" "$URL/sub_definitely_not_here")
check "get unknown id is 404" 404 "$code"

# Cleanup.
code=$(curl -sS -o /dev/null -w '%{http_code}' -X DELETE "${HDR[@]}" "$URL/$ID")
check "delete is 204" 204 "$code"

printf '\nresult: %d passed, %d failed\n' "$pass" "$fail"
[[ "$fail" -eq 0 ]]
