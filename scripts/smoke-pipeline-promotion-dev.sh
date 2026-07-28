#!/usr/bin/env bash
# Smoke tests for pipeline promotion (CYB-3914).
# Usage:
#   source scripts/dev-backend-env.sh
#   bash scripts/smoke-pipeline-promotion-dev.sh
set -euo pipefail

BASE="${BASE:-http://localhost:8080}"
TOKEN="${TOKEN:-dev-token}"
IDEMPOTENCY_KEY="smoke-$(date +%s)-$$"
PASS=0
FAIL=0

log_pass() { echo "  ✓ $1"; PASS=$((PASS + 1)); }
log_fail() { echo "  ✗ $1 — $2"; FAIL=$((FAIL + 1)); }

api() {
	local method="$1" path="$2" body="${3:-}" extra_args=("${@:4}")
	local curl_args=( -s -w '\n%{http_code}' -X "$method" -H "X-Databrew-Token: $TOKEN" -H "Content-Type: application/json" )
	if [[ -n "$body" ]]; then
		curl_args+=( -d "$body" )
	fi
	curl_args+=("${extra_args[@]}" "$BASE/api/v1$path")
	curl "${curl_args[@]}"
}

check_status() {
	local label="$1" expected="$2" status="$3" body="$4"
	if [[ "$status" == "$expected" ]]; then
		log_pass "$label"
	else
		log_fail "$label" "expected HTTP $expected got $status: $(echo "$body" | head -c 200)"
	fi
}

echo "=== Pipeline Promotion Smoke ==="
echo "BASE=$BASE"
echo ""

# ── 1. Create a dev pipeline ──
echo "--- 1. Create dev template ---"
PIPE_NAME="promotion-smoke-${IDEMPOTENCY_KEY}"
RESULT=$(api "POST" "/pipelines" "{\"name\":\"$PIPE_NAME\",\"pipeline\":{\"nodes\":[{\"id\":\"step1\",\"component\":{\"name\":\"echo\",\"image\":\"alpine:3.18@sha256:11dcd0588b9b0a8c8b1e9dfb2a5ab72b98a77548adee262d73e5a3eeb3e0ffa0\"}}],\"edges\":[]}}")
STATUS=$(echo "$RESULT" | tail -1)
BODY=$(echo "$RESULT" | sed '$d')
TEMPLATE_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])" 2>/dev/null || echo "")
if [[ -n "$TEMPLATE_ID" ]]; then
	log_pass "created dev template $TEMPLATE_ID"
else
	log_fail "create dev template" "no id: $(echo "$BODY" | head -c 200)"
	exit 1
fi

# ── 2. Get promotion plan ──
echo "--- 2. Promotion plan ---"
RESULT=$(api "POST" "/pipelines/${TEMPLATE_ID}/promotion-plan" '{"target":"prod"}')
STATUS=$(echo "$RESULT" | tail -1)
BODY=$(echo "$RESULT" | sed '$d')
if [[ "$STATUS" == "200" ]]; then
	READY=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('ready',False))" 2>/dev/null || echo "False")
	log_pass "promotion plan (ready=$READY)"
else
	log_fail "promotion plan" "HTTP $STATUS"
fi

# ── 3. Promote ──
echo "--- 3. Promote to prod ---"
RESULT=$(api "POST" "/pipelines/${TEMPLATE_ID}/promote" "")
STATUS=$(echo "$RESULT" | tail -1)
BODY=$(echo "$RESULT" | sed '$d')
if [[ "$STATUS" == "201" ]]; then
	PROD_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])" 2>/dev/null || echo "")
	PROD_SCOPE=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('scope',''))" 2>/dev/null || echo "")
	if [[ "$PROD_SCOPE" == "prod" ]]; then
		log_pass "promoted to prod (id=$PROD_ID, scope=$PROD_SCOPE)"
	else
		log_fail "promote" "expected scope=prod got scope=$PROD_SCOPE"
	fi
else
	log_fail "promote" "HTTP $STATUS: $(echo "$BODY" | head -c 200)"
fi

# ── 4. Plan for non-existent template → 404 ──
echo "--- 4. Plan non-existent template ---"
RESULT=$(api "POST" "/pipelines/nonexistent-id/promotion-plan" '{"target":"prod"}')
STATUS=$(echo "$RESULT" | tail -1)
BODY=$(echo "$RESULT" | sed '$d')
check_status "plan non-existent" "404" "$STATUS" "$BODY"

# ── 5. Cleanup dev template (prod stays read-only) ──
echo "--- 5. Cleanup ---"
RESULT=$(api "DELETE" "/pipelines/${TEMPLATE_ID}" "")
STATUS=$(echo "$RESULT" | tail -1)
check_status "cleanup dev" "204" "$STATUS" "$(echo "$RESULT" | sed '$d')"

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
[[ "$FAIL" -eq 0 ]] && echo "✓ All promotion smoke tests passed" || echo "✗ Some tests failed"
exit "$FAIL"
