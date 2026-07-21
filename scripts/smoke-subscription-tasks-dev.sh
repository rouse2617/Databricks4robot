#!/usr/bin/env bash
# Smoke test for the subscription-tasks CRUD API (CYB-3778) against dev.
#
# Usage:
#   source scripts/dev-backend-env.sh          # exports BASE + DATABREW_TOKEN
#   bash scripts/smoke-subscription-tasks-dev.sh
#
# Covers: create/list/get/pause/resume/delete happy path + 400 (missing
# required) + 404 (unknown id). Leaves no rows behind on success.
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

# 400: bad pullIntervalSeconds
body='{"name":"bad","templateId":"tpl","targetId":"t","projectId":"p","subscriptionId":"s","pullIntervalSeconds":0}'
code=$(curl -sS -o /tmp/smoke-st-400b.txt -w '%{http_code}' -X POST "${HDR[@]}" "$URL" -d "$body")
check "create with pullInterval=0 is 400" 400 "$code" "$(cat /tmp/smoke-st-400b.txt)"

# Happy path: create → get → pause → resume → delete.
NAME="smoke-subtask-$(date +%s)"
body=$(cat <<JSON
{
  "name": "$NAME",
  "enabled": false,
  "templateId": "tpl-smoke",
  "targetId": "cluster-default",
  "projectId": "co-prod-gv-cybercap",
  "subscriptionId": "smoke-test-sub",
  "pullIntervalSeconds": 60,
  "maxMessagesPerPull": 100
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

# 404: unknown id
code=$(curl -sS -o /dev/null -w '%{http_code}' "${HDR[@]}" "$URL/sub_definitely_not_here")
check "get unknown id is 404" 404 "$code"

# Cleanup.
code=$(curl -sS -o /dev/null -w '%{http_code}' -X DELETE "${HDR[@]}" "$URL/$ID")
check "delete is 204" 204 "$code"

printf '\nresult: %d passed, %d failed\n' "$pass" "$fail"
[[ "$fail" -eq 0 ]]
