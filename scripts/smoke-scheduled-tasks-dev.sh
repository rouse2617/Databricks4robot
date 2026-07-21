#!/usr/bin/env bash
# Smoke test for the scheduled-tasks CRUD API (CYB-3744) against dev.
#
# Usage:
#   source scripts/dev-backend-env.sh          # exports BASE + DATABREW_TOKEN
#   bash scripts/smoke-scheduled-tasks-dev.sh
#
# Covers: create/list/get/pause/resume/run-now/delete happy path + 400 (bad
# triggerMode) + 404 (unknown id). Leaves no rows behind on success.
set -euo pipefail

: "${BASE:?source scripts/dev-backend-env.sh first (BASE not set)}"
: "${DATABREW_TOKEN:?set DATABREW_TOKEN}"

URL="${BASE}/api/v1/scheduled-tasks"
HDR=( -H "X-Databrew-Token: ${DATABREW_TOKEN}" -H "Content-Type: application/json" )

pass=0; fail=0
check() { # $1=name, $2=expected-code, $3=actual-code, $4=optional-body
    if [[ "$3" == "$2" ]]; then
        printf '  ✅ %s → %s\n' "$1" "$3"; pass=$((pass+1))
    else
        printf '  ❌ %s → %s (want %s)\n     body: %s\n' "$1" "$3" "$2" "${4:-}"; fail=$((fail+1))
    fi
}

# 400: bad triggerMode
body=$(cat <<'JSON'
{"name":"smoke-bad","templateId":"tpl","targetId":"cluster-default","sourceType":"rest","triggerMode":"cron"}
JSON
)
code=$(curl -sS -o /tmp/smoke-st-400.txt -w '%{http_code}' -X POST "${HDR[@]}" "$URL" -d "$body")
check "create with bad triggerMode is 400" 400 "$code" "$(cat /tmp/smoke-st-400.txt)"

# 400: missing required
code=$(curl -sS -o /tmp/smoke-st-400b.txt -w '%{http_code}' -X POST "${HDR[@]}" "$URL" -d '{"name":"x"}')
check "create missing required is 400" 400 "$code" "$(cat /tmp/smoke-st-400b.txt)"

# Happy path: create → get → pause → resume → run-now → delete.
NAME="smoke-scheduled-$(date +%s)"
body=$(cat <<JSON
{
  "name": "$NAME",
  "enabled": false,
  "templateId": "tpl-smoke",
  "targetId": "cluster-default",
  "sourceType": "rest",
  "sourceConfig": {"base_url":"https://example.invalid","path":"/x","id_path":"data[].id"},
  "triggerMode": "incremental",
  "triggerConfig": {"intervalSeconds": 3600}
}
JSON
)
resp=$(curl -sS -w '\nHTTPCODE:%{http_code}' -X POST "${HDR[@]}" "$URL" -d "$body")
code="${resp##*HTTPCODE:}"; json="${resp%HTTPCODE:*}"
check "create happy path is 201" 201 "$code" "$json"
ID=$(printf '%s' "$json" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("id",""))')
if [[ -z "$ID" ]]; then echo "  ❌ could not extract id from create body"; fail=$((fail+1)); ID="sched_unknown"; fi

code=$(curl -sS -o /dev/null -w '%{http_code}' "${HDR[@]}" "$URL/$ID")
check "get by id is 200" 200 "$code"

code=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "${HDR[@]}" "$URL/$ID/pause")
check "pause is 200" 200 "$code"

code=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "${HDR[@]}" "$URL/$ID/resume")
check "resume is 200" 200 "$code"

code=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "${HDR[@]}" "$URL/$ID/run-now")
check "run-now is 202" 202 "$code"

# 404: unknown id
code=$(curl -sS -o /dev/null -w '%{http_code}' "${HDR[@]}" "$URL/sched_definitely_not_here")
check "get unknown id is 404" 404 "$code"

# Cleanup.
code=$(curl -sS -o /dev/null -w '%{http_code}' -X DELETE "${HDR[@]}" "$URL/$ID")
check "delete is 204" 204 "$code"

printf '\nresult: %d passed, %d failed\n' "$pass" "$fail"
[[ "$fail" -eq 0 ]]
