#!/bin/bash
# Smoke test for CYB-3267 Bug 1: GET /assets/:id/actions must return 200 (not
# 500) once an asset has an action. Before the fix, scanAction dropped task_id
# so the row scan failed with "number of field descriptions must equal number
# of destinations, got 22 and 21".
# Usage:
#   BASE=https://cyber-databrew-dev.cyberorigin.ai TOKEN=dev-token \
#   bash scripts/smoke-asset-actions-dev.sh [mcap_file_id]

set -euo pipefail

: "${BASE:?BASE is required}"
: "${TOKEN:?TOKEN is required}"
hdr=(-H "X-Databrew-Token: $TOKEN" -H "Content-Type: application/json")

MID="${1:-}"
if [[ -z "$MID" ]]; then
  MID=$(curl -s "${hdr[@]}" "$BASE/api/v1/mcap-files?page=1&page_size=1" \
    | python3 -c "import json,sys;d=json.load(sys.stdin);i=(d.get('items') or d.get('mcap_files') or []);print(i[0]['mcap_file_id'] if i else '')")
fi
[[ -n "$MID" ]] || { echo "no mcap_file_id"; exit 1; }
echo "Using mcap_file_id $MID"

echo "1️⃣  POST /assets (raw_mcap) → create asset"
AID=$(curl -s "${hdr[@]}" -X POST "$BASE/api/v1/assets" \
  -d "{\"asset_type\":\"raw_mcap\",\"mcap_file_id\":\"$MID\",\"reviewer\":\"cyb3267-smoke\",\"start_timestamp_ns\":1000000000,\"end_timestamp_ns\":3000000000}" \
  | python3 -c "import json,sys;print(json.load(sys.stdin).get('asset_id',''))")
[[ -n "$AID" ]] || { echo "   ❌ asset create failed"; exit 1; }
echo "   asset_id=$AID"

echo "2️⃣  POST /assets/$AID/actions → create an action"
CODE=$(curl -s -o /dev/null -w "%{http_code}" "${hdr[@]}" -X POST "$BASE/api/v1/assets/$AID/actions" \
  -d '{"start_ns":1000000000,"end_ns":2000000000,"primary_label":"cyb3267_probe","source_type":"human","source_name":"smoke"}')
[[ "$CODE" == "201" || "$CODE" == "200" ]] && echo "   ✅ HTTP $CODE" || { echo "   ❌ HTTP $CODE (action create)"; exit 1; }

echo "3️⃣  GET /assets/$AID/actions → expect 200 (was 500 before fix)"
OUT=$(curl -s -w "\n%{http_code}" "${hdr[@]}" "$BASE/api/v1/assets/$AID/actions")
CODE=$(echo "$OUT" | tail -1)
BODY=$(echo "$OUT" | sed '$d')
[[ "$CODE" == "200" ]] && echo "   ✅ HTTP $CODE" || { echo "   ❌ HTTP $CODE — body: $BODY"; exit 1; }

echo "4️⃣  response has the action (total >= 1)"
echo "$BODY" | python3 -c "import json,sys;d=json.load(sys.stdin);t=d.get('total',0);sys.exit(0 if t>=1 else 1)" \
  && echo "   ✅ total >= 1" || { echo "   ❌ no actions returned"; exit 1; }

echo "✅ asset actions read smoke passed (Bug 1 fixed)."
