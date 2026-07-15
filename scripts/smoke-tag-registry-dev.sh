#!/bin/bash
# Smoke test for CYB-3246 Phase 2: admin managed tag-registry CRUD.
# - create (201) -> list contains key -> update (200) -> duplicate (409)
# - non-admin token (401) -> delete (200)
# Auth: AdminTokenOrAdminRole. In dev with ADMIN_TOKEN unset the DatabrewToken
# (X-Databrew-Token) passes; if dev sets ADMIN_TOKEN, export it and it is sent
# via X-Admin-Token too.
# Usage:
#   BASE=https://cyber-databrew-dev.cyberorigin.ai TOKEN=dev-token \
#   [ADMIN_TOKEN=...] bash scripts/smoke-tag-registry-dev.sh

set -euo pipefail

: "${BASE:?BASE is required}"
: "${TOKEN:?TOKEN is required}"
ADMIN_TOKEN="${ADMIN_TOKEN:-$TOKEN}"

# Send both credentials so whichever mode dev runs in is satisfied.
auth=(-H "X-Databrew-Token: $TOKEN" -H "X-Admin-Token: $ADMIN_TOKEN" -H "Content-Type: application/json")

KEY="smoke_severity_$$"

cleanup() {
  curl -s -o /dev/null "${auth[@]}" -X DELETE "$BASE/api/v1/admin/tag-registry/$KEY" || true
}
trap cleanup EXIT

echo "1️⃣  POST create '$KEY' (enum) → expect 201"
CODE=$(curl -s -o /dev/null -w "%{http_code}" "${auth[@]}" -X POST "$BASE/api/v1/admin/tag-registry" \
  -d "{\"key\":\"$KEY\",\"description\":\"smoke\",\"type\":\"enum\",\"values\":[\"critical\",\"high\",\"low\"]}")
[[ "$CODE" == "201" ]] && echo "   ✅ HTTP $CODE" || { echo "   ❌ HTTP $CODE (expected 201)"; exit 1; }

echo "2️⃣  GET list → contains '$KEY'"
curl -s "${auth[@]}" "$BASE/api/v1/admin/tag-registry" \
  | python3 -c "import json,sys;keys=[i['key'] for i in json.load(sys.stdin).get('items',[])];sys.exit(0 if '$KEY' in keys else 1)" \
  && echo "   ✅ present" || { echo "   ❌ not in list"; exit 1; }

echo "3️⃣  PATCH update '$KEY' (add value 'info') → expect 200"
CODE=$(curl -s -o /dev/null -w "%{http_code}" "${auth[@]}" -X PATCH "$BASE/api/v1/admin/tag-registry/$KEY" \
  -d '{"type":"enum","values":["critical","high","low","info"]}')
[[ "$CODE" == "200" ]] && echo "   ✅ HTTP $CODE" || { echo "   ❌ HTTP $CODE (expected 200)"; exit 1; }

echo "4️⃣  POST duplicate '$KEY' → expect 409"
CODE=$(curl -s -o /dev/null -w "%{http_code}" "${auth[@]}" -X POST "$BASE/api/v1/admin/tag-registry" \
  -d "{\"key\":\"$KEY\",\"type\":\"string\"}")
[[ "$CODE" == "409" ]] && echo "   ✅ HTTP $CODE" || { echo "   ❌ HTTP $CODE (expected 409)"; exit 1; }

echo "5️⃣  POST with bogus token → expect 401"
CODE=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "X-Databrew-Token: not-a-real-token" -H "X-Admin-Token: not-a-real-token" -H "Content-Type: application/json" \
  -X POST "$BASE/api/v1/admin/tag-registry" -d '{"key":"x","type":"string"}')
[[ "$CODE" == "401" ]] && echo "   ✅ HTTP $CODE" || { echo "   ❌ HTTP $CODE (expected 401)"; exit 1; }

echo "6️⃣  DELETE '$KEY' → expect 200"
CODE=$(curl -s -o /dev/null -w "%{http_code}" "${auth[@]}" -X DELETE "$BASE/api/v1/admin/tag-registry/$KEY")
[[ "$CODE" == "200" ]] && echo "   ✅ HTTP $CODE" || { echo "   ❌ HTTP $CODE (expected 200)"; exit 1; }

echo "✅ tag-registry admin CRUD smoke passed."
