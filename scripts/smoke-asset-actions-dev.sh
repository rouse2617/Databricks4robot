#!/bin/bash
# Smoke test for CYB-3268: /assets/:id/actions 4 methods unified on the assets
# table as first-class assets (asset_type='action'). Covers the full happy path
# (POST → GET → PATCH → GET → DELETE → GET) plus error paths (unregistered label
# → 422, duration < 1ms → 422, cross-parent PATCH → 404, missing aid → 404).
#
# Also still guards CYB-3267 Bug 1 (GET must be 200, not 500).
#
# Usage:
#   BASE=https://cyber-databrew-dev.cyberorigin.ai TOKEN=dev-token \
#   bash scripts/smoke-asset-actions-dev.sh

set -euo pipefail

: "${BASE:?BASE is required}"
: "${TOKEN:?TOKEN is required}"
hdr=(-H "X-Databrew-Token: $TOKEN" -H "Content-Type: application/json")

jqpy() { python3 -c "import json,sys;d=json.load(sys.stdin);print($1)"; }

echo "1️⃣  pick an existing segment (action requires a segment parent)"
SEG=$(curl -s "${hdr[@]}" -X POST "$BASE/api/v1/queries/run" \
  -d '{"schema_version":"v1","scope":{"resource":"assets"},"where":{"pred":{"field":"asset_type","op":"eq","value":"segment"}},"page":{"page":1,"page_size":1}}' \
  | jqpy "d.get('items',[{}])[0].get('asset_id','')")
[[ -n "$SEG" ]] || { echo "   ❌ no segment found"; exit 1; }
echo "   segment=$SEG"

echo "2️⃣  POST /assets/$SEG/actions → create first-class action (v2 body)"
# primary_label must be in action_label_registry; "inspect" is registered on dev.
CREATE=$(curl -s "${hdr[@]}" -X POST "$BASE/api/v1/assets/$SEG/actions" \
  -d '{"start_timestamp_ns":1000000000,"end_timestamp_ns":2000000000,"metadata":{"primary_label":"inspect","labels":["inspect"],"description":"smoke","source_type":"human","source_name":"smoke"}}')
AID=$(echo "$CREATE" | jqpy "d.get('asset_id','')")
ATYPE=$(echo "$CREATE" | jqpy "d.get('asset_type','')")
PARENT=$(echo "$CREATE" | jqpy "d.get('parent_asset_id','')")
PLABEL=$(echo "$CREATE" | jqpy "d.get('metadata',{}).get('primary_label','')")
[[ -n "$AID" && "$ATYPE" == "action" && "$PARENT" == "$SEG" && "$PLABEL" == "inspect" ]] \
  && echo "   ✅ created $AID (asset_type=action, parent=$SEG, metadata.primary_label=inspect)" \
  || { echo "   ❌ unexpected create response: $CREATE"; exit 1; }

echo "3️⃣  GET /assets/$SEG/actions → 200 (CYB-3267 Bug 1) and contains the new action"
OUT=$(curl -s -w "\n%{http_code}" "${hdr[@]}" "$BASE/api/v1/assets/$SEG/actions")
CODE=$(echo "$OUT" | tail -1); BODY=$(echo "$OUT" | sed '$d')
[[ "$CODE" == "200" ]] || { echo "   ❌ HTTP $CODE — body: $BODY"; exit 1; }
echo "$BODY" | python3 -c "import json,sys;d=json.load(sys.stdin);sys.exit(0 if any(i.get('asset_id')=='$AID' and i.get('asset_type')=='action' for i in d.get('items',[])) else 1)" \
  && echo "   ✅ GET 200; action present as first-class asset row" \
  || { echo "   ❌ created action not found in list: $BODY"; exit 1; }

echo "4️⃣  PATCH /assets/$SEG/actions/$AID → merge metadata"
PATCHED=$(curl -s "${hdr[@]}" -X PATCH "$BASE/api/v1/assets/$SEG/actions/$AID" \
  -d '{"metadata":{"description":"smoke-updated"}}')
DESC=$(echo "$PATCHED" | jqpy "d.get('metadata',{}).get('description','')")
[[ "$DESC" == "smoke-updated" ]] \
  && echo "   ✅ metadata.description updated" \
  || { echo "   ❌ PATCH did not update metadata: $PATCHED"; exit 1; }

echo "5️⃣  error paths"
# unregistered label → 422
C=$(curl -s -o /dev/null -w "%{http_code}" "${hdr[@]}" -X POST "$BASE/api/v1/assets/$SEG/actions" \
  -d '{"start_timestamp_ns":1000000000,"end_timestamp_ns":2000000000,"metadata":{"primary_label":"definitely_not_a_label"}}')
[[ "$C" == "422" ]] && echo "   ✅ unregistered label → 422" || { echo "   ❌ unregistered label → $C (want 422)"; exit 1; }
# duration < 1ms → 422
C=$(curl -s -o /dev/null -w "%{http_code}" "${hdr[@]}" -X POST "$BASE/api/v1/assets/$SEG/actions" \
  -d '{"start_timestamp_ns":1000000000,"end_timestamp_ns":1000000500,"metadata":{"primary_label":"inspect"}}')
[[ "$C" == "422" ]] && echo "   ✅ duration<1ms → 422" || { echo "   ❌ duration<1ms → $C (want 422)"; exit 1; }
# cross-parent PATCH (use the action's own id as the parent) → 404
C=$(curl -s -o /dev/null -w "%{http_code}" "${hdr[@]}" -X PATCH "$BASE/api/v1/assets/$AID/actions/$AID" \
  -d '{"metadata":{"description":"x"}}')
[[ "$C" == "404" ]] && echo "   ✅ cross-parent PATCH → 404" || { echo "   ❌ cross-parent PATCH → $C (want 404)"; exit 1; }
# missing aid PATCH → 404
C=$(curl -s -o /dev/null -w "%{http_code}" "${hdr[@]}" -X PATCH "$BASE/api/v1/assets/$SEG/actions/zzzzzzzz" \
  -d '{"metadata":{"description":"x"}}')
[[ "$C" == "404" ]] && echo "   ✅ missing aid PATCH → 404" || { echo "   ❌ missing aid PATCH → $C (want 404)"; exit 1; }

echo "6️⃣  DELETE /assets/$SEG/actions/$AID → 204, then gone from GET"
C=$(curl -s -o /dev/null -w "%{http_code}" "${hdr[@]}" -X DELETE "$BASE/api/v1/assets/$SEG/actions/$AID")
[[ "$C" == "204" ]] || { echo "   ❌ DELETE → $C (want 204)"; exit 1; }
GONE=$(curl -s "${hdr[@]}" "$BASE/api/v1/assets/$SEG/actions" \
  | jqpy "sum(1 for i in d.get('items',[]) if i.get('asset_id')=='$AID')")
[[ "$GONE" == "0" ]] && echo "   ✅ deleted; no longer in list" || { echo "   ❌ action still present after delete"; exit 1; }

echo "✅ asset actions first-class smoke passed (CYB-3268 happy + error paths)."
