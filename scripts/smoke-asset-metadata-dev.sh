#!/bin/bash
# Smoke test for CYB-3203 asset metadata endpoint
# Usage:
#   source scripts/dev-backend-env.sh
#   bash scripts/smoke-asset-metadata-dev.sh <asset_id>

set -euo pipefail

: "${BASE:?BASE is required. source scripts/dev-backend-env.sh first}"
: "${TOKEN:?TOKEN is required. source scripts/dev-backend-env.sh first}"

ASSET_ID="${1:-91022781}"

echo "Testing GET /api/v1/assets/{id}/metadata against dev"
echo "  BASE=$BASE"
echo "  ASSET_ID=$ASSET_ID"
echo ""

# Test 1: Happy path - existing asset with metadata
echo "1️⃣  Happy path (existing asset: $ASSET_ID)"
RESPONSE=$(curl -s -H "X-Databrew-Token: $TOKEN" "$BASE/api/v1/assets/$ASSET_ID/metadata")
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "X-Databrew-Token: $TOKEN" "$BASE/api/v1/assets/$ASSET_ID/metadata")

if [ "$HTTP_CODE" == "200" ]; then
    echo "   ✅ HTTP $HTTP_CODE"
    echo "   Response keys: $(echo "$RESPONSE" | python3 -c "import json,sys; d=json.load(sys.stdin); print(', '.join(d.keys()))" 2>/dev/null || echo '(parse error)')"

    # Verify response structure
    HAS_ASSET_ID=$(echo "$RESPONSE" | python3 -c "import json,sys; print('asset_id' in json.load(sys.stdin))" 2>/dev/null || echo 'false')
    HAS_MCAP_META=$(echo "$RESPONSE" | python3 -c "import json,sys; print('mcap_metadata' in json.load(sys.stdin))" 2>/dev/null || echo 'false')

    if [ "$HAS_ASSET_ID" == "True" ] && [ "$HAS_MCAP_META" == "True" ]; then
        echo "   ✅ Response structure valid (asset_id, mcap_metadata present)"
    else
        echo "   ❌ Response structure incomplete"
        exit 1
    fi
else
    echo "   ❌ HTTP $HTTP_CODE (expected 200)"
    exit 1
fi

echo ""

# Test 2: 404 path - non-existent asset
echo "2️⃣  404 path (non-existent asset: 00000000)"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "X-Databrew-Token: $TOKEN" "$BASE/api/v1/assets/00000000/metadata")

if [ "$HTTP_CODE" == "404" ]; then
    echo "   ✅ HTTP $HTTP_CODE"
else
    echo "   ❌ HTTP $HTTP_CODE (expected 404)"
    exit 1
fi

echo ""
echo "✅ All smoke tests passed!"
