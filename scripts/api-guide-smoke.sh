#!/usr/bin/env bash
# Smoke tests aligned with docs/review/api-guide.md (curl examples).
# Usage:
#   BASE=http://localhost:8080 TOKEN=dev-token bash scripts/api-guide-smoke.sh
#   RUN_WRITES=1 BASE=http://host:8080 TOKEN=dev-token bash scripts/api-guide-smoke.sh
#
# HTTPS + Google IAP (e.g. dev Gateway): also set IAP_TOKEN to an OIDC JWT whose
# audience is the IAP OAuth 2.0 client ID (same value as GCPBackendPolicy iap.clientID).
#   BASE=https://api-cyber-databrew-dev.cyberorigin.ai TOKEN=<DATABREW_TOKEN> \
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
	API_HDR=( -H "X-Databrew-Token: ${TOKEN}" -H "Content-Type: application/json" )
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

expect_code_get() {
	local name="$1" path="$2" expected="$3"
	local raw code body
	raw=$(curl -sS --max-time 25 -w "\n%{http_code}" "${API_HDR[@]}" "${BASE}${path}" 2>/dev/null) || raw=$'\n000'
	code=$(echo "$raw" | tail -n1)
	body=$(echo "$raw" | sed '$d')
	RESP_CODE="$code"
	RESP_BODY="$body"
	if [[ "$code" == "$expected" ]]; then
		ok "$name"
	else
		bad "$name (expected ${expected})"
	fi
	echo "$body"
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

post_json() {
	local name="$1" path="$2" data="$3"
	local raw
	raw=$(curl -sS --max-time 30 -w "\n%{http_code}" -X POST "${API_HDR[@]}" "$BASE$path" -d "$data" 2>/dev/null) || raw=$'\n000'
	RESP_CODE=$(echo "$raw" | tail -n1)
	RESP_BODY=$(echo "$raw" | sed '$d')
	if [[ "$RESP_CODE" =~ ^2 ]]; then
		PASS=$((PASS + 1))
		echo "  OK  $name" >&2
	else
		FAIL=$((FAIL + 1))
		echo "  FAIL $name (HTTP ${RESP_CODE})" >&2
		echo "$RESP_BODY" | head -c 400 >&2
		echo >&2
	fi
	echo "$RESP_BODY"
}

expect_code_post() {
	local name="$1" path="$2" data="$3" expected="$4"
	local raw code body
	raw=$(curl -sS --max-time 30 -w "\n%{http_code}" -X POST "${API_HDR[@]}" "$BASE$path" -d "$data" 2>/dev/null) || raw=$'\n000'
	code=$(echo "$raw" | tail -n1)
	body=$(echo "$raw" | sed '$d')
	RESP_CODE="$code"
	RESP_BODY="$body"
	if [[ "$code" == "$expected" ]]; then
		ok "$name"
	else
		bad "$name (expected ${expected})"
	fi
	echo "$body"
}

expect_json_number() {
	local name="$1" body="$2" field="$3" expected="$4"
	local got
	got=$(echo "$body" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('${field}', ''))" 2>/dev/null || echo "")
	if [[ "$got" == "$expected" ]]; then
		ok "$name"
	else
		RESP_CODE="json"
		RESP_BODY="$body"
		bad "$name expected ${field}=${expected}, got ${got:-<empty>}"
	fi
}

echo "=== api-guide smoke === BASE=$BASE"
if [[ "$BASE" == https://* ]] && [[ -z "${IAP_TOKEN:-}" ]]; then
	echo "  NOTE: HTTPS BASE without IAP_TOKEN — if the host uses IAP, expect 302/401; set IAP_TOKEN (OIDC, audience = IAP OAuth client ID)."
fi
echo ""

echo "--- health (no X-Databrew-Token; IAP Bearer optional) ---"
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
if [[ "$RESP_CODE" == "200" ]]; then
	if echo "$RESP_BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); items=d.get('items', []); assert isinstance(items, list); assert all('table_name' in i and 'row_count' in i for i in items)" 2>/dev/null; then
		ok "lakehouse/tables response shape"
		if echo "$RESP_BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); sys.exit(0 if any(i.get('table_name') == 'silver_asset_events_current' for i in d.get('items', [])) else 1)" 2>/dev/null; then
			ok "lakehouse/tables silver_asset_events_current visible"
		else
			echo "  OK  lakehouse/tables silver_asset_events_current absent — tolerated until Silver export job has run"
			PASS=$((PASS + 1))
		fi
	else
		bad "lakehouse/tables response shape"
	fi
fi
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
get "search sync status" "/api/v1/search/sync-status"
post "queries run (structured)" "/api/v1/queries/run" '{"schema_version":"v1","mode":"structured","scope":{"resource":"assets"},"page":{"page":1,"page_size":5}}' >/dev/null
post "queries run (include_history)" "/api/v1/queries/run?include_history=true" '{"schema_version":"v1","scope":{"resource":"assets","include_history":true},"page":{"page":1,"page_size":5}}' >/dev/null
post "queries run (keyword)" "/api/v1/queries/run" '{"schema_version":"v1","mode":"keyword","scope":{"resource":"assets"},"where":{"pred":{"field":"_fulltext","op":"ilike","value":"warehouse"}},"page":{"page":1,"page_size":5}}' >/dev/null
get "deliveries list" "/api/v1/deliveries?page=1&page_size=5"
get "mcap-files list" "/api/v1/mcap-files?page=1&page_size=5"

echo ""
echo "--- § algo-runs (CYB-1018/CYB-1123) ---"
get "GET algo-runs list (page/page_size)" "/api/v1/algo-runs?page=1&page_size=5"
if [[ "${RUN_WRITES:-0}" == "1" ]]; then
	RUN_ID=$(python3 -c "import secrets,string; a=string.ascii_letters+string.digits; print(''.join(secrets.choice(a) for _ in range(16)))")
	post "POST algo-runs" "/api/v1/algo-runs" "{\"run_id\":\"${RUN_ID}\",\"algo_name\":\"hand_track\",\"algo_version\":\"2.0\",\"algo_kind\":\"processing\",\"triggered_by\":\"manual:api-guide-smoke\"}" >/dev/null
	expect_code_post "POST algo-runs duplicate run_id -> 409" "/api/v1/algo-runs" "{\"run_id\":\"${RUN_ID}\",\"algo_name\":\"hand_track\",\"algo_version\":\"2.0\",\"algo_kind\":\"processing\",\"triggered_by\":\"manual:api-guide-smoke\"}" "409" >/dev/null
	get "GET algo-runs/{id}" "/api/v1/algo-runs/${RUN_ID}"
	post "POST algo-runs start" "/api/v1/algo-runs/${RUN_ID}/start" "{}" >/dev/null
	get "GET algo-runs affected-assets" "/api/v1/algo-runs/${RUN_ID}/affected-assets"
	post "POST algo-runs finish" "/api/v1/algo-runs/${RUN_ID}/finish" "{\"status\":\"ok\",\"assets_processed\":1,\"assets_succeeded\":1,\"assets_failed\":0}" >/dev/null
else
	echo "  skip algo-runs write smoke — set RUN_WRITES=1 to exercise POST/GET /algo-runs"
fi

echo ""
echo "--- § customers (CYB-1014) ---"
get "GET customers list" "/api/v1/customers?limit=5"
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
LIST_RAW=$(curl -sS --max-time 20 -X POST "${API_HDR[@]}" "${BASE}/api/v1/queries/run" -d '{"schema_version":"v1","mode":"structured","scope":{"resource":"assets"},"page":{"page":1,"page_size":5}}' || true)
AID=$(echo "$LIST_RAW" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d['items'][0]['asset_id'])" 2>/dev/null || echo "")
LAID=$(echo "$LIST_RAW" | python3 -c "import sys,json;d=json.load(sys.stdin);print(next((i.get('logical_asset_id') for i in d.get('items', []) if i.get('logical_asset_id')), ''))" 2>/dev/null || echo "")
if [[ -n "$AID" ]]; then
	get "asset by id" "/api/v1/assets/${AID}"
	get "asset provenance" "/api/v1/assets/${AID}/provenance"
	raw=$(curl -sS -N --max-time 3 -w "\n%{http_code}" "${API_HDR[@]}" "${BASE}/api/v1/assets/${AID}/events/stream" 2>/dev/null || true)
	RESP_CODE=$(echo "$raw" | tail -n1)
	RESP_BODY=$(echo "$raw" | sed '$d')
	if [[ "$RESP_CODE" == "200" ]]; then ok "asset events stream SSE"; else bad "asset events stream SSE"; fi
	raw=$(curl -sS --max-time 10 -w "\n%{http_code}" "${API_HDR[@]}" -H "Last-Event-ID: not-a-number" "${BASE}/api/v1/assets/${AID}/events/stream" 2>/dev/null || echo $'\n000')
	RESP_CODE=$(echo "$raw" | tail -n1)
	RESP_BODY=$(echo "$raw" | sed '$d')
	if [[ "$RESP_CODE" == "400" ]]; then ok "asset events stream invalid Last-Event-ID -> 400"; else bad "asset events stream invalid Last-Event-ID expected 400"; fi
else
	echo "  skip GET asset/{id} — could not parse list"
fi
if [[ -n "$LAID" ]]; then
	get "logical asset ratings history" "/api/v1/logical-assets/${LAID}/ratings-history"
else
	echo "  skip logical-assets ratings-history — could not parse logical_asset_id"
fi
expect_code_get "logical asset ratings history invalid id -> 400" "/api/v1/logical-assets/not-valid/ratings-history" "400" >/dev/null

echo ""
echo "--- §2.5.1 audit search (CYB-1097) ---"
get "audit search latest" "/api/v1/audit/search?limit=5"
expect_code_get "audit search invalid time_from -> 400" "/api/v1/audit/search?time_from=not-rfc3339" "400" >/dev/null

echo ""
echo "--- §2.5.2 audit lineage search (CYB-1098) ---"
if [[ -n "${AID:-}" ]]; then
	get "audit lineage search by asset" "/api/v1/audit/lineage-search?asset_id=${AID}&direction=both&depth=2"
else
	echo "  skip audit lineage success — could not parse asset id from list"
fi
LINEAGE_AID="${AID:-asset001}"
expect_code_get "audit lineage invalid direction -> 400" "/api/v1/audit/lineage-search?asset_id=${LINEAGE_AID}&direction=sideways" "400" >/dev/null

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

	if [[ -n "${CUST_ID:-}" && -n "${NEW_AID:-}" ]]; then
		echo ""
		echo "--- §3 delivery C2 + idempotency (CYB-1123/CYB-1124) ---"
		DRAFT_RAW=$(post_json "POST deliveries/draft" "/api/v1/deliveries/draft" "{\"customer_id\":\"${CUST_ID}\",\"asset_ids\":[\"${NEW_AID}\"],\"note\":\"smoke draft\"}")
		DRAFT_ID=$(echo "$DRAFT_RAW" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('delivery_id',''))" 2>/dev/null || echo "")
		if [[ -n "$DRAFT_ID" ]]; then
			ITEMS_RAW=$(post_json "POST deliveries/{id}/items duplicate does not inflate count" "/api/v1/deliveries/${DRAFT_ID}/items" "{\"asset_ids\":[\"${NEW_AID}\"]}")
			expect_json_number "delivery add-items duplicate asset_count stays 1" "$ITEMS_RAW" "asset_count" "1"
			EXPECTED_REV=$(echo "$ITEMS_RAW" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('version',''))" 2>/dev/null || echo "")
			if [[ -n "$EXPECTED_REV" ]]; then
				COMMIT_RAW=$(post_json "POST deliveries/{id}/commit" "/api/v1/deliveries/${DRAFT_ID}/commit" "{\"expected_revision\":${EXPECTED_REV},\"approved_by\":\"api-guide-smoke\"}")
				COMMIT_ID=$(echo "$COMMIT_RAW" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('delivery_id',''))" 2>/dev/null || echo "")
				if [[ -n "$COMMIT_ID" ]]; then
					post "POST deliveries/{id}/ack" "/api/v1/deliveries/${COMMIT_ID}/ack" '{"acknowledged_by":"api-guide-smoke"}' >/dev/null
				else
					echo "  WARN delivery commit smoke skipped ack: missing delivery_id"
				fi
			else
				echo "  WARN delivery commit smoke skipped: missing expected revision from add-items"
			fi
		else
			echo "  WARN delivery draft smoke skipped: missing delivery_id"
		fi
		IDEM_KEY="smoke-$(python3 -c "import secrets,string; print(''.join(secrets.choice(string.ascii_lowercase+string.digits) for _ in range(10)))")"
		DELIVERY_JSON_1="{\"customer_id\":\"${CUST_ID}\",\"asset_ids\":[\"${NEW_AID}\"],\"note\":\"idempotency smoke 1\"}"
		DELIVERY_JSON_2="{\"customer_id\":\"${CUST_ID}\",\"asset_ids\":[\"${NEW_AID}\"],\"note\":\"idempotency smoke 2\"}"
		raw=$(curl -sS --max-time 30 -w "\n%{http_code}" -X POST "${API_HDR[@]}" -H "Idempotency-Key: ${IDEM_KEY}" "$BASE/api/v1/deliveries" -d "$DELIVERY_JSON_1" 2>/dev/null || echo $'\n000')
		RESP_CODE=$(echo "$raw" | tail -n1)
		RESP_BODY=$(echo "$raw" | sed '$d')
		if [[ "$RESP_CODE" =~ ^2 ]]; then ok "POST deliveries idempotency seed"; else bad "POST deliveries idempotency seed"; fi
		raw=$(curl -sS --max-time 30 -w "\n%{http_code}" -X POST "${API_HDR[@]}" -H "Idempotency-Key: ${IDEM_KEY}" "$BASE/api/v1/deliveries" -d "$DELIVERY_JSON_2" 2>/dev/null || echo $'\n000')
		RESP_CODE=$(echo "$raw" | tail -n1)
		RESP_BODY=$(echo "$raw" | sed '$d')
		if [[ "$RESP_CODE" == "409" ]]; then ok "POST deliveries same idem key different payload -> 409"; else bad "POST deliveries same idem key different payload expected 409"; fi
		# §3.5 delivery cancel/retry (CYB-1135)
		echo ""
		echo "--- §3.5 delivery cancel/retry (CYB-1135) ---"
		CANCEL_DRAFT_RAW=$(post_json "POST deliveries/draft for cancel smoke" "/api/v1/deliveries/draft" "{\"customer_id\":\"${CUST_ID}\",\"asset_ids\":[\"${NEW_AID}\"],\"note\":\"smoke cancel test\"}")
		CANCEL_DRAFT_ID=$(echo "$CANCEL_DRAFT_RAW" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('delivery_id',''))" 2>/dev/null || echo "")
		if [[ -n "$CANCEL_DRAFT_ID" ]]; then
			CANCEL_ITEMS_RAW=$(post_json "POST deliveries/{id}/items for cancel smoke" "/api/v1/deliveries/${CANCEL_DRAFT_ID}/items" "{\"asset_ids\":[\"${NEW_AID}\"]}")
			CANCEL_REV=$(echo "$CANCEL_ITEMS_RAW" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('version',''))" 2>/dev/null || echo "")
			if [[ -n "$CANCEL_REV" ]]; then
				CANCEL_COMMIT_RAW=$(post_json "POST deliveries/{id}/commit for cancel smoke" "/api/v1/deliveries/${CANCEL_DRAFT_ID}/commit" "{\"expected_revision\":${CANCEL_REV},\"approved_by\":\"api-guide-smoke\"}")
				CANCEL_COMMIT_ID=$(echo "$CANCEL_COMMIT_RAW" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('delivery_id',''))" 2>/dev/null || echo "")
				if [[ -n "$CANCEL_COMMIT_ID" ]]; then
					# Happy: cancel delivered
					CANCEL_OUT=$(expect_code_post "POST deliveries/{id}/cancel delivered" "/api/v1/deliveries/${CANCEL_COMMIT_ID}/cancel" '{"cancelled_by":"api-guide-smoke","cancel_reason":"smoke test"}' "200")
					cancel_status=$(echo "$CANCEL_OUT" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('status',''))" 2>/dev/null || echo "")
					if [[ "$cancel_status" == "cancelled" ]]; then ok "cancel delivered → status=cancelled"; else bad "cancel delivered expected status=cancelled, got ${cancel_status:-<empty>}"; fi

					# Happy: retry cancelled
					RETRY_OUT=$(expect_code_post "POST deliveries/{id}/retry cancelled" "/api/v1/deliveries/${CANCEL_COMMIT_ID}/retry" '{}' "201")
					retry_status=$(echo "$RETRY_OUT" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('status',''))" 2>/dev/null || echo "")
					RETRY_ID=$(echo "$RETRY_OUT" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('delivery_id',''))" 2>/dev/null || echo "")
					if [[ "$retry_status" == "pending" ]]; then ok "retry cancelled → status=pending"; else bad "retry cancelled expected status=pending, got ${retry_status:-<empty>}"; fi

					# Error: ack pending → 422
					if [[ -n "${RETRY_ID:-}" ]]; then
						expect_code_post "POST deliveries/{id}/ack pending → 422" "/api/v1/deliveries/${RETRY_ID}/ack" '{"acknowledged_by":"api-guide-smoke"}' "422"
					fi
				else
					echo "  WARN cancel/retry smoke skipped: missing commit delivery_id"
				fi
			else
				echo "  WARN cancel/retry smoke skipped: missing expected revision"
			fi
		else
			echo "  WARN cancel/retry smoke skipped: missing draft_id"
		fi

		# Error: cancel non-existent delivery → 404
		expect_code_post "POST deliveries/{id}/cancel non-existent → 404" "/api/v1/deliveries/00000000-0000-0000-0000-000000000000/cancel" '{"cancelled_by":"test","cancel_reason":"test"}' "404"
	fi
fi

echo ""
echo "=== done: ${PASS} passed, ${FAIL} failed ==="
[[ "$FAIL" -eq 0 ]]
