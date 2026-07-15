#!/bin/bash
# Smoke test for CYB-3232: PATCH /assets/:id supports files (merge) + storage_uri/thumb_uri.
# Usage:
#   source scripts/dev-backend-env.sh
#   bash scripts/smoke-asset-patch-files-dev.sh [asset_id]

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

KEY="algo_input_forward_stereo"
VAL="gs://smoke-bucket/forward_stereo/$AID.mp4"

echo "1️⃣  PATCH files → expect 200"
CODE=$(curl -s -o /dev/null -w "%{http_code}" "${hdr[@]}" -X PATCH "$BASE/api/v1/assets/$AID" \
  -d "{\"files\":{\"$KEY\":\"$VAL\"},\"thumb_uri\":\"gs://smoke-bucket/thumb/$AID.jpg\"}")
[[ "$CODE" == "200" ]] && echo "   ✅ HTTP $CODE" || { echo "   ❌ HTTP $CODE"; exit 1; }

echo "2️⃣  GET → files.$KEY persisted"
GOT=$(curl -s "${hdr[@]}" "$BASE/api/v1/assets/$AID" \
  | python3 -c "import json,sys;d=json.load(sys.stdin);print((d.get('files') or {}).get('$KEY',''),'|',d.get('thumb_uri',''))")
echo "   files.$KEY | thumb_uri = $GOT"
echo "$GOT" | grep -q "$VAL" && echo "   ✅ merged value present" || { echo "   ❌ value not persisted"; exit 1; }

echo "✅ PATCH files smoke passed."
