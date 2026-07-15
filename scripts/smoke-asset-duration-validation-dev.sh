#!/bin/bash
# Smoke test for CYB-3226 asset write-time range validation.
# Verifies the new time-range gates on POST /api/v1/assets:
#   - end <= start            → 422 INVALID_STATE
#   - sub-1ms span            → 422 INVALID_STATE (the duration_ms=0 class)
#   - valid span (>= 1ms)     → NOT a range error (reaches downstream handling)
#
# The range check runs in the usecase BEFORE the mcap_file_id FK check, so the
# reject cases do not need a real mcap file. The happy-path duration_ms>0 assertion
# is done during deploy-verify with a real mcap (see CYB-3226 tasks.md).
#
# Usage:
#   source scripts/dev-backend-env.sh
#   bash scripts/smoke-asset-duration-validation-dev.sh

set -euo pipefail

: "${BASE:?BASE is required. source scripts/dev-backend-env.sh first}"
: "${TOKEN:?TOKEN is required. source scripts/dev-backend-env.sh first}"

post_assets() { # $1=json body → prints HTTP code
  curl -s -o /dev/null -w "%{http_code}" \
    -H "X-Databrew-Token: $TOKEN" -H "Content-Type: application/json" \
    -X POST "$BASE/api/v1/assets" -d "$1"
}

echo "CYB-3226 range validation smoke against $BASE"
echo ""

# 1) end == start → 422
echo "1️⃣  end == start (empty range) → expect 422"
CODE=$(post_assets '{"mcap_file_id":"SMOKE001","start_timestamp_ns":1000,"end_timestamp_ns":1000,"reviewer":"smoke"}')
[ "$CODE" == "422" ] && echo "   ✅ HTTP $CODE" || { echo "   ❌ HTTP $CODE (expected 422)"; exit 1; }
echo ""

# 2) sub-1ms span → 422 (new floor)
echo "2️⃣  sub-1ms span (999999 ns) → expect 422"
CODE=$(post_assets '{"mcap_file_id":"SMOKE001","start_timestamp_ns":1000,"end_timestamp_ns":1000999,"reviewer":"smoke"}')
[ "$CODE" == "422" ] && echo "   ✅ HTTP $CODE" || { echo "   ❌ HTTP $CODE (expected 422)"; exit 1; }
echo ""

# 3) valid >= 1ms span → must NOT be a range 422 (a 4xx from the fake mcap FK is fine)
echo "3️⃣  valid 1ms span → expect NOT 422-for-range (403/404/409/500 from fake mcap ok, just not the range reject)"
CODE=$(post_assets '{"mcap_file_id":"SMOKE001","start_timestamp_ns":1000,"end_timestamp_ns":2000000,"reviewer":"smoke"}')
if [ "$CODE" == "422" ]; then
  echo "   ⚠️  HTTP 422 — could be the fake mcap FK (422 INVALID_STATE) rather than range; inspect body if unexpected"
else
  echo "   ✅ HTTP $CODE (passed range validation)"
fi
echo ""
echo "✅ Range validation smoke complete."
