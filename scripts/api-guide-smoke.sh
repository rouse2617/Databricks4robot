#!/usr/bin/env bash
# Smoke tests aligned with docs/review/api-guide.md (curl examples).
# Usage:
#   BASE=http://localhost:8080 TOKEN=dev-token bash scripts/api-guide-smoke.sh
#   RUN_WRITES=1 BASE=http://host:8080 TOKEN=dev-token bash scripts/api-guide-smoke.sh
#
# HTTPS + Google IAP (e.g. dev Gateway): also set IAP_TOKEN to an OIDC JWT whose
# audience is the IAP OAuth 2.0 client ID (same value as GCPBackendPolicy iap.clientID).
#   BASE=https://api-cyber-databrew-dev.cyberorigin.ai TOKEN=<GRACE_TOKEN> \
#   IAP_TOKEN="$(gcloud auth print-identity-token --impersonate-service-account=... \
#     --audiences=XXXX.apps.googleusercontent.com)" \
#   bash scripts/api-guide-smoke.sh
#
# Requires: curl, python3 (json extract). Optional RUN_WRITES=1 for POST mcap-files + assets.
set -euo pipefail

BASE="${BASE:-http://localhost:8080}"
TOKEN="${TOKEN:-dev-token}"
declare -a API_HDR=()
declare -a HEALTH_HDR=()

refresh_headers() {
	API_HDR=( -H "X-Grace-Token: ${TOKEN}" -H "Content-Type: application/json" )
	HEALTH_HDR=()
	if [[ -n "${IAP_TOKEN:-}" ]]; then
		API_HDR+=( -H "Authorization: Bearer ${IAP_TOKEN}" )
		HEALTH_HDR=( -H "Authorization: Bearer ${IAP_TOKEN}" )
	fi
}

refresh_headers

RESP_BODY=""
RESP_CODE=""
PASS=0
FAIL=0

ok() { PASS=$((PASS + 1)); echo "  OK  $1"; }
bad() {
	local label="$1"
	FAIL=$((FAIL + 1))
	echo "  FAIL $label (HTTP ${RESP_CODE})"
	echo "$RESP_BODY" | head -c 400
	echo
}

get() {
	local name="$1" path="$2"
	local raw
	raw=$(curl -sS --max-time 25 -w "\n%{http_code}" "${API_HDR[@]}" "${BASE}${path}" 2>/dev/null) || raw=$'\n000'
	RESP_CODE=$(echo "$raw" | tail -n1)
	RESP_BODY=$(echo "$raw" | sed '$d')
	if [[ "$RESP_CODE" =~ ^2 ]]; then ok "$name"; else bad "$name"; fi
}

warn_get() {
	local name="$1" path="$2"
	local raw
	raw=$(curl -sS --max-time 25 -w "\n%{http_code}" "${API_HDR[@]}" "${BASE}${path}" 2>/dev/null) || raw=$'\n000'
	RESP_CODE=$(echo "$raw" | tail -n1)
	RESP_BODY=$(echo "$raw" | sed '$d')
	if [[ "$RESP_CODE" =~ ^2 ]]; then ok "$name"; else echo "  WARN $name (HTTP ${RESP_CODE}) — optional / needs full Iceberg MVP"; fi
}

get_report_or_skip() {
	local raw
	raw=$(curl -sS --max-time 25 -w "\n%{http_code}" "${API_HDR[@]}" "${BASE}/api/v1/lakehouse/report" 2>/dev/null) || raw=$'\n000'
	RESP_CODE=$(echo "$raw" | tail -n1)
	RESP_BODY=$(echo "$raw" | sed '$d')
	if [[ "$RESP_CODE" == "200" ]]; then ok "lakehouse/report"; elif [[ "$RESP_CODE" == "404" ]]; then echo "  OK  lakehouse/report (404 — run make iceberg-mvp locally if you need file)"; PASS=$((PASS + 1)); else bad "lakehouse/report"; fi
}

post() {
	local name="$1" path="$2" data="$3"
	local raw
	raw=$(curl -sS --max-time 30 -w "\n%{http_code}" -X POST "${API_HDR[@]}" "$BASE$path" -d "$data" 2>/dev/null) || raw=$'\n000'
	RESP_CODE=$(echo "$raw" | tail -n1)
	RESP_BODY=$(echo "$raw" | sed '$d')
	if [[ "$RESP_CODE" =~ ^2 ]]; then ok "$name"; else bad "$name"; fi
	echo "$RESP_BODY"
}

echo "=== api-guide smoke === BASE=$BASE"
if [[ "$BASE" == https://* ]] && [[ -z "${IAP_TOKEN:-}" ]]; then
	echo "  NOTE: HTTPS BASE without IAP_TOKEN — if the host uses IAP, expect 302/401; set IAP_TOKEN (OIDC, audience = IAP OAuth client ID)."
fi
echo ""

echo "--- health (no X-Grace-Token; IAP Bearer optional) ---"
if [[ -n "${IAP_TOKEN:-}" ]]; then
	raw=$(curl -sS --max-time 15 -w "\n%{http_code}" -H "Authorization: Bearer ${IAP_TOKEN}" "${BASE}/healthz" 2>/dev/null) || raw=$'\n000'
else
	raw=$(curl -sS --max-time 15 -w "\n%{http_code}" "${BASE}/healthz" 2>/dev/null) || raw=$'\n000'
fi
RESP_CODE=$(echo "$raw" | tail -n1)
RESP_BODY=$(echo "$raw" | sed '$d')
if [[ "$RESP_CODE" == "200" ]]; then ok "GET /healthz"; else bad "GET /healthz"; fi

echo ""
echo "--- § Lakehouse / Trino 验证 ---"
get "lakehouse/status" "/api/v1/lakehouse/status"
get "lakehouse/tables" "/api/v1/lakehouse/tables"
get_report_or_skip
warn_get "lakehouse/training-assets" "/api/v1/lakehouse/training-assets?snapshot_id=mvp_hand_tracking_quality_v1"
warn_get "lakehouse/recompute-candidates" "/api/v1/lakehouse/recompute-candidates?algo_key=hand_tracking@1.2.0&target_version=1.3.0"
# Trino-dependent; dev often has TRINO_DISABLED (503) — same as training-assets / recompute-candidates.
warn_get "lakehouse/tag-timeline" "/api/v1/lakehouse/tag-timeline?tag_key=quality"
warn_get "lakehouse/quality-distribution" "/api/v1/lakehouse/quality-distribution?window=30d"
warn_get "lakehouse/customer-replay" "/api/v1/lakehouse/customer-replay?customer_id=urn:grace:customer:A"

echo ""
echo "--- § 注册表 ---"
get "algo-registry" "/api/v1/algo-registry"
get "tag-registry" "/api/v1/tag-registry"

echo ""
echo "--- § 资产 / 搜索 / 交付 / mcap-files ---"
post "queries run (structured)" "/api/v1/queries/run" '{"schema_version":"v1","mode":"structured","scope":{"resource":"assets"},"page":{"page":1,"page_size":5}}' >/dev/null
post "queries run (include_history)" "/api/v1/queries/run?include_history=true" '{"schema_version":"v1","scope":{"resource":"assets","include_history":true},"page":{"page":1,"page_size":5}}' >/dev/null
post "queries run (keyword)" "/api/v1/queries/run" '{"schema_version":"v1","mode":"keyword","scope":{"resource":"assets"},"where":{"pred":{"field":"_fulltext","op":"ilike","value":"warehouse"}},"page":{"page":1,"page_size":5}}' >/dev/null
get "deliveries list" "/api/v1/deliveries?page=1&page_size=5"
get "mcap-files list" "/api/v1/mcap-files?page=1&page_size=5"

echo ""
echo "--- § customers (CYB-1014) ---"
if [[ "${RUN_WRITES:-0}" == "1" ]]; then
	CUST_ID="smoke$(python3 -c "import secrets,string; print(''.join(secrets.choice(string.ascii_lowercase+string.digits) for _ in range(8)))")"
	post "POST customers" "/api/v1/customers" "{\"customer_id\":\"${CUST_ID}\",\"display_name\":\"api-guide smoke\"}" >/dev/null
	get "GET customers/{id}" "/api/v1/customers/${CUST_ID}"
	get "deliveries by customer_id" "/api/v1/deliveries?page=1&page_size=5&customer_id=${CUST_ID}"
else
	echo "  skip customer write smoke — set RUN_WRITES=1 to exercise POST/GET /customers"
fi
get "metrics registry" "/api/v1/metrics/registry"

echo ""
echo "--- optional GET asset by id (first from list) ---"
LIST_RAW=$(curl -sS --max-time 20 -X POST "${API_HDR[@]}" "${BASE}/api/v1/queries/run" -d '{"schema_version":"v1","mode":"structured","scope":{"resource":"assets"},"page":{"page":1,"page_size":1}}' || true)
AID=$(echo "$LIST_RAW" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d['items'][0]['asset_id'])" 2>/dev/null || echo "")
if [[ -n "$AID" ]]; then
	get "asset by id" "/api/v1/assets/${AID}"
else
	echo "  skip GET asset/{id} — could not parse list"
fi

if [[ "${RUN_WRITES:-0}" == "1" ]]; then
	echo ""
	echo "--- RUN_WRITES=1 — §1.1 mcap-files + assets (unique ids) ---"
	MCAP_ID=$(python3 -c "import secrets,string; a=string.ascii_letters+string.digits; print(''.join(secrets.choice(a) for _ in range(8)))")
	HASH=$(openssl rand -hex 16 2>/dev/null || python3 -c "import secrets; print(secrets.token_hex(16))")
	MCAP_JSON=$(cat <<EOF
{
  "mcap_file_id": "${MCAP_ID}",
  "gcs_path": "gs://api-guide-smoke/path/file.mcap",
  "raw_hash_md5": "${HASH}",
  "ingest_state": "summarized",
  "size_bytes": 1048576,
  "file_duration_ms": 60000,
  "start_timestamp_ns": 1700000000000000000,
  "end_timestamp_ns": 1700000060000000000,
  "channel_count": 12,
  "chunk_count": 5,
  "vendor_id": "smoke",
  "device_id": "smoke-1",
  "scene_id": "indoor",
  "owner": "api-guide-smoke"
}
EOF
)
	post "POST mcap-files" "/api/v1/mcap-files" "$MCAP_JSON" >/dev/null
	ASSET_JSON=$(cat <<EOF
{
  "mcap_file_id": "${MCAP_ID}",
  "start_timestamp_ns": 1700000000000000000,
  "end_timestamp_ns": 1700000060000000000,
  "reviewer": "smoke",
  "owner": "api-guide-smoke",
  "type": "task_demo",
  "tags": { "priority": "low", "quality": "good", "scene": "indoor" }
}
EOF
)
	post "POST assets" "/api/v1/assets" "$ASSET_JSON" >/dev/null

	echo ""
	echo "--- §2.4 multi-source tags (CYB-1015) ---"
	NEW_AID=$(curl -sS --max-time 20 -X POST "${API_HDR[@]}" "${BASE}/api/v1/queries/run" -d '{"schema_version":"v1","mode":"structured","scope":{"resource":"assets"},"page":{"page":1,"page_size":1}}' \
		| python3 -c "import sys,json;d=json.load(sys.stdin);print(d['items'][0]['asset_id'])" 2>/dev/null || echo "")
	if [[ -n "$NEW_AID" ]]; then
		# human upsert (requires source_name)
		post "POST tags human"      "/api/v1/assets/${NEW_AID}/tags" '{"key":"quality","value":"good","source_type":"human","source_name":"smoke"}' >/dev/null
		# rule_engine upsert (requires source_name + source_version)
		post "POST tags rule_engine" "/api/v1/assets/${NEW_AID}/tags" '{"key":"quality","value":"good","source_type":"rule_engine","source_name":"qc","source_version":"1.0"}' >/dev/null
		# human upsert missing source_name → 422 TAG_SOURCE_INVALID
		raw=$(curl -sS --max-time 25 -w "\n%{http_code}" -X POST "${API_HDR[@]}" "${BASE}/api/v1/assets/${NEW_AID}/tags" -d '{"key":"quality","value":"good","source_type":"human"}' || echo $'\n000')
		code="${raw##*$'\n'}"
		if [[ "$code" == "422" ]]; then ok "POST tags human (no source_name) → 422"; else bad "POST tags human (no source_name) → expected 422 got $code"; fi
		# source-scoped delete (only human row)
		raw=$(curl -sS --max-time 25 -w "\n%{http_code}" -X DELETE "${API_HDR[@]}" "${BASE}/api/v1/assets/${NEW_AID}/tags/quality?source_type=human" || echo $'\n000')
		code="${raw##*$'\n'}"
		if [[ "$code" == "200" ]]; then ok "DELETE tags?source_type=human"; else bad "DELETE tags?source_type=human got $code"; fi
		# detail surface
		get "GET asset tags_detailed" "/api/v1/assets/${NEW_AID}"
	else
		echo "  skip multi-source tags smoke — no asset id available"
	fi
fi

echo ""
echo "=== done: ${PASS} passed, ${FAIL} failed ==="
[[ "$FAIL" -eq 0 ]]
