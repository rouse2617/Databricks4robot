#!/bin/bash
# Smoke test for CYB-3246 Phase 1: open-vocabulary tag validation.
# - Unregistered string key is accepted (200) and persisted.
# - Registered enum key with an invalid value is still rejected (422).
# - Registered enum key with a valid value still passes (200, no regression).
# Usage:
#   source scripts/dev-backend-env.sh
#   bash scripts/smoke-tag-openvocab-dev.sh [asset_id]

set -euo pipefail

: "${BASE:?BASE is required. source scripts/dev-backend-env.sh first}"
: "${TOKEN:?TOKEN is required. source scripts/dev-backend-env.sh first}"

hdr=(-H "X-Databrew-Token: $TOKEN" -H "Content-Type: application/json")

# Pick an asset: arg, else first from queries/run.
AID="${1:-}"
if [[ -z "$AID" ]]; then
  AID=$(curl -s "${hdr[@]}" -X POST "$BASE/api/v1/queries/run" \
    -d '{"schema_version":"v1","scope":{"resource":"assets"},"page":{"page":1,"page_size":1}}' \
    | python3 -c "import json,sys;d=json.load(sys.stdin);print((d.get('items') or [{}])[0].get('asset_id',''))")
fi
[[ -n "$AID" ]] || { echo "no asset found"; exit 1; }
echo "Using asset $AID"

UNREG_KEY="smoke_desc"
UNREG_VAL="开放词汇冒烟 城市街道 下午"

echo "1️⃣  POST unregistered key '$UNREG_KEY' → expect 200 (open vocabulary)"
CODE=$(curl -s -o /dev/null -w "%{http_code}" "${hdr[@]}" -X POST "$BASE/api/v1/assets/$AID/tags" \
  -d "{\"key\":\"$UNREG_KEY\",\"value\":\"$UNREG_VAL\",\"source_type\":\"human\",\"source_name\":\"smoke\"}")
[[ "$CODE" == "200" ]] && echo "   ✅ HTTP $CODE" || { echo "   ❌ HTTP $CODE (expected 200)"; exit 1; }

echo "2️⃣  GET → unregistered tag persisted"
GOT=$(curl -s "${hdr[@]}" "$BASE/api/v1/assets/$AID" \
  | python3 -c "import json,sys;d=json.load(sys.stdin);print((d.get('tags') or {}).get('$UNREG_KEY',''))")
echo "   tags.$UNREG_KEY = $GOT"
[[ "$GOT" == "$UNREG_VAL" ]] && echo "   ✅ persisted" || { echo "   ❌ not persisted"; exit 1; }

echo "3️⃣  POST registered enum 'priority'='urgent' (invalid) → expect 422"
CODE=$(curl -s -o /dev/null -w "%{http_code}" "${hdr[@]}" -X POST "$BASE/api/v1/assets/$AID/tags" \
  -d '{"key":"priority","value":"urgent","source_type":"human","source_name":"smoke"}')
[[ "$CODE" == "422" ]] && echo "   ✅ HTTP $CODE" || { echo "   ❌ HTTP $CODE (expected 422)"; exit 1; }

echo "4️⃣  POST registered enum 'priority'='high' (valid) → expect 200 (no regression)"
CODE=$(curl -s -o /dev/null -w "%{http_code}" "${hdr[@]}" -X POST "$BASE/api/v1/assets/$AID/tags" \
  -d '{"key":"priority","value":"high","source_type":"human","source_name":"smoke"}')
[[ "$CODE" == "200" ]] && echo "   ✅ HTTP $CODE" || { echo "   ❌ HTTP $CODE (expected 200)"; exit 1; }

echo "✅ open-vocabulary tag smoke passed."
