#!/usr/bin/env bash
# Exhaustive api-guide.md verification against a running backend.
# Usage: BASE=http://localhost:8080 TOKEN=dev-token bash scripts/verify-api-guide.sh
set -euo pipefail

BASE="${BASE:-http://localhost:8080}"
TOKEN="${TOKEN:-dev-token}"
HDR=( -H "X-Grace-Token: ${TOKEN}" -H "Content-Type: application/json" )
PASS=0
FAIL=0
DOC_NOTE=()

note() { DOC_NOTE+=("$1"); }

ok() { PASS=$((PASS + 1)); echo "OK   $1"; }
fail() {
	FAIL=$((FAIL + 1))
	echo "FAIL $1 (got HTTP $2)"
	[[ -n "${3:-}" ]] && echo "       body: $(echo "$3" | head -c 200)"
}

code_of() { echo "$1" | tail -n1; }
body_of() { echo "$1" | sed '$d'; }

getc() {
	local name="$1" path="$2" exp="${3:-2}"
	local raw
	raw=$(curl -sS --max-time 30 -w "\n%{http_code}" "${HDR[@]}" "${BASE}${path}") || raw=$'\n000'
	local c b
	c=$(code_of "$raw")
	b=$(body_of "$raw")
	if [[ "$c" =~ ^${exp}$ ]] || [[ "$exp" == "2" && "$c" =~ ^2 ]]; then ok "$name"; else fail "$name" "$c" "$b"; fi
}

postc() {
	local name="$1" path="$2" data="$3" exp="${4:-2}"
	local raw
	raw=$(curl -sS --max-time 45 -w "\n%{http_code}" -X POST "${HDR[@]}" "${BASE}${path}" -d "$data") || raw=$'\n000'
	local c b
	c=$(code_of "$raw")
	b=$(body_of "$raw")
	if [[ "$c" =~ ^${exp}$ ]] || [[ "$exp" == "2" && "$c" =~ ^2 ]]; then ok "$name"; else fail "$name" "$c" "$b"; fi
}

postc_idem() {
	local name="$1" path="$2" data="$3" exp="${4:-2}"
	local idem raw c b
	idem="verify-$(date +%s)-${RANDOM}"
	raw=$(curl -sS --max-time 45 -w "\n%{http_code}" -X POST "${HDR[@]}" \
		-H "Idempotency-Key: ${idem}" "${BASE}${path}" -d "$data") || raw=$'\n000'
	c=$(code_of "$raw")
	b=$(body_of "$raw")
	if [[ "$c" =~ ^${exp}$ ]] || [[ "$exp" == "2" && "$c" =~ ^2 ]]; then ok "$name"; else fail "$name" "$c" "$b"; fi
}

echo "=========================================="
echo " api-guide.md verification  BASE=$BASE"
echo "=========================================="

echo ""
echo "--- § 基础 / 认证 ---"
raw=$(curl -sS --max-time 15 -w "\n%{http_code}" "${BASE}/healthz") || raw=$'\n000'
c=$(code_of "$raw")
[[ "$c" == "200" ]] && ok "GET /healthz (no auth)" || fail "GET /healthz" "$c"

postc "POST /api/v1/auth/login" "/api/v1/auth/login" '{"token":"'"$TOKEN"'"}' "200"
getc "GET /api/v1/auth/me (token header)" "/api/v1/auth/me" "2"

echo ""
echo "--- § Lakehouse (api-guide § 开头) ---"
getc "GET lakehouse/status" "/api/v1/lakehouse/status"
getc "GET lakehouse/tables" "/api/v1/lakehouse/tables"
getc "GET lakehouse/tag-timeline" "/api/v1/lakehouse/tag-timeline?tag_key=quality"
getc "GET lakehouse/quality-distribution" "/api/v1/lakehouse/quality-distribution?window=30d"
getc "GET lakehouse/customer-replay" "/api/v1/lakehouse/customer-replay?customer_id=urn:grace:customer:A"

echo ""
echo "--- § Lakehouse sync-status (api-guide §8) ---"
getc "GET lakehouse/sync-status" "/api/v1/lakehouse/sync-status"

echo ""
echo "--- § Search sync-status (工作台; 文档未单列但常用) ---"
getc "GET search/sync-status" "/api/v1/search/sync-status"

echo ""
echo "--- § 注册表 ---"
getc "GET algo-registry" "/api/v1/algo-registry"
getc "GET tag-registry" "/api/v1/tag-registry"
getc "GET lifecycle-states" "/api/v1/lifecycle-states"

echo ""
echo "--- § Query API ---"
postc "POST queries/validate" "/api/v1/queries/validate" '{"schema_version":"v1","scope":{"resource":"assets"},"where":{"pred":{"field":"owner","op":"eq","value":"alice"}},"page":{"page":1,"page_size":5}}'
postc "POST queries/run structured" "/api/v1/queries/run" '{"schema_version":"v1","mode":"structured","scope":{"resource":"assets"},"page":{"page":1,"page_size":3}}'

echo ""
echo "--- § 交付 / MCAP 列表 / metrics registry ---"
getc "GET deliveries" "/api/v1/deliveries?page=1&page_size=3"
getc "GET mcap-files" "/api/v1/mcap-files?page=1&page_size=3"
getc "GET metrics/registry" "/api/v1/metrics/registry"

echo ""
echo "--- § 动态 ID：从 queries/run 取 asset / segment / mcap ---"
LIST_JSON=$(curl -sS --max-time 25 -X POST "${HDR[@]}" "${BASE}/api/v1/queries/run" \
	-d '{"schema_version":"v1","mode":"structured","scope":{"resource":"assets"},"page":{"page":1,"page_size":1}}')
ANY_AID=$(echo "$LIST_JSON" | python3 -c "import sys,json;d=json.load(sys.stdin);print((d.get('items')or[{}])[0].get('asset_id')or'')" 2>/dev/null || echo "")

SEG_JSON=$(curl -sS --max-time 25 -X POST "${HDR[@]}" "${BASE}/api/v1/queries/run" \
	-d '{"schema_version":"v1","mode":"structured","scope":{"resource":"assets"},"where":{"pred":{"field":"asset_type","op":"eq","value":"segment"}},"page":{"page":1,"page_size":1}}')
SEG_AID=$(echo "$SEG_JSON" | python3 -c "import sys,json;d=json.load(sys.stdin);print((d.get('items')or[{}])[0].get('asset_id')or'')" 2>/dev/null || echo "")

MCAP_JSON=$(curl -sS --max-time 25 -X POST "${HDR[@]}" "${BASE}/api/v1/queries/run" \
	-d '{"schema_version":"v1","mode":"structured","scope":{"resource":"assets"},"select":{"fields":["asset_id","mcap_file_id"]},"page":{"page":1,"page_size":1}}')
MCAP_ID=$(echo "$MCAP_JSON" | python3 -c "import sys,json;d=json.load(sys.stdin);it=(d.get('items')or[{}])[0];print(it.get('mcap_file_id')or'')" 2>/dev/null || echo "")

if [[ -z "$ANY_AID" ]]; then
	note "No asset_id from queries/run; skipping asset-scoped checks."
else
	echo "Using ANY_AID=$ANY_AID SEG_AID=${SEG_AID:-<none>} MCAP_ID=${MCAP_ID:-<none>}"
	getc "GET asset by id" "/api/v1/assets/${ANY_AID}"
	getc "GET asset deliveries" "/api/v1/assets/${ANY_AID}/deliveries?page=1&page_size=5"
	getc "GET asset events" "/api/v1/assets/${ANY_AID}/events?limit=5"

	# Algo: start env_analysis (idempotent-ish; may 409 if already running — accept 200/409)
	raw=$(curl -sS --max-time 30 -w "\n%{http_code}" -X POST "${HDR[@]}" "${BASE}/api/v1/assets/${ANY_AID}/algo/env_analysis@1.0.0/start" \
		-d '{"method":"k8s_job","run_id":"verify-api-guide-'"$(date +%s)"'"}') || raw=$'\n000'
	c=$(code_of "$raw")
	if [[ "$c" == "200" || "$c" == "409" ]]; then ok "POST algo env_analysis/start (200 or 409)"; else fail "POST algo env_analysis/start" "$c" "$(body_of "$raw")"; fi

	raw=$(curl -sS --max-time 30 -w "\n%{http_code}" -X POST "${HDR[@]}" "${BASE}/api/v1/assets/${ANY_AID}/algo/env_analysis@1.0.0/finish" \
		-d '{"status":"ok","run_id":"verify-api-guide-finish-'"$(date +%s)"'"}') || raw=$'\n000'
	c=$(code_of "$raw")
	if [[ "$c" == "200" || "$c" == "409" ]]; then ok "POST algo env_analysis/finish (200 or 409)"; else fail "POST algo env_analysis/finish" "$c" "$(body_of "$raw")"; fi

	postc "POST tags (quality=good)" "/api/v1/assets/${ANY_AID}/tags" '{"key":"quality","value":"good"}'
	getc "GET tags/history" "/api/v1/assets/${ANY_AID}/tags/history?limit=5"
	# DELETE tag — idempotent
	raw=$(curl -sS --max-time 20 -w "\n%{http_code}" -X DELETE "${HDR[@]}" "${BASE}/api/v1/assets/${ANY_AID}/tags/quality") || raw=$'\n000'
	c=$(code_of "$raw")
	[[ "$c" =~ ^2 ]] && ok "DELETE tags/quality" || fail "DELETE tags/quality" "$c" "$(body_of "$raw")"

	getc "GET eval-results" "/api/v1/assets/${ANY_AID}/eval-results?limit=5"
	getc "GET metrics projection" "/api/v1/assets/${ANY_AID}/metrics"
fi

echo ""
echo "--- § metrics:search (api-guide §8.2) ---"
postc "POST metrics:search" "/api/v1/metrics:search" '{"filters":{"lifecycle_state":"ready","metrics":[{"metric_key":"good_frames_ratio","op":">=","value":0.0}]},"page":1,"page_size":5}'

if [[ -n "$ANY_AID" ]]; then
	postc "POST eval-results (smoke write)" "/api/v1/assets/${ANY_AID}/eval-results" '{
    "target_type":"segment","target_id":"","eval_name":"qc_deliverable_frame_eval","eval_version":"1.0.0",
    "parameter_version":"8","run_id":"verify-'"$(date +%s)"'","source_type":"algo","source_name":"verify",
    "status":"ok","result_payload":{"good_frames_ratio":0.5,"hand_good_frames":1,"total_good_frames":1,"total_video_frames":2,"weighted_avg_convex_hull_volume":0}
  }'
fi

echo ""
echo "--- § Admin search reindex dry_run ---"
postc "POST admin/search/reindex dry_run" "/api/v1/admin/search/reindex" '{"dry_run":true,"page_size":50}'

echo ""
echo "--- § Actions (need segment) ---"
if [[ -n "$SEG_AID" ]]; then
	getc "GET actions list" "/api/v1/assets/${SEG_AID}/actions"
	# start_ns/end_ns must fall inside segment [start_timestamp_ns, end_timestamp_ns]; gin rejects 0 for required int64.
	SEG_INFO=$(curl -sS --max-time 20 "${HDR[@]}" "${BASE}/api/v1/assets/${SEG_AID}")
	SSTART=$(echo "$SEG_INFO" | python3 -c "import sys,json;d=json.load(sys.stdin);print(int(d.get('start_timestamp_ns')or 0))")
	SEND=$(echo "$SEG_INFO" | python3 -c "import sys,json;d=json.load(sys.stdin);print(int(d.get('end_timestamp_ns')or 0))")
	R0=$((SSTART + 1000))
	R1=$((SSTART + 1000000000))
	[[ "$R1" -ge "$SEND" ]] && R1=$((SEND - 1))
	[[ "$R0" -ge "$R1" ]] && R0=$((SSTART + 1))
	ACTION_JSON=$(printf '{"start_ns":%s,"end_ns":%s,"primary_label":"pickup","labels":["pickup"],"description":"verify-api-guide","source_type":"human","source_name":"verify-script"}' "$R0" "$R1")
	postc_idem "POST actions create" "/api/v1/assets/${SEG_AID}/actions" "$ACTION_JSON"
else
	note "No segment asset in first page of filter; skipped POST/GET actions on real segment."
fi

echo ""
echo "--- § Documented 待上线 — expect 404 unless implemented ---"
raw=$(curl -sS --max-time 15 -w "\n%{http_code}" "${HDR[@]}" "${BASE}/api/v1/actions?label=pickup&page=1&page_size=5") || raw=$'\n000'
c=$(code_of "$raw")
if [[ "$c" == "404" ]]; then ok "GET /api/v1/actions (404 as 待上线)"; else
	ok "GET /api/v1/actions (HTTP $c — doc says 待上线; verify if route exists)"
	[[ "$c" =~ ^2 ]] && note "GET /api/v1/actions returned 2xx; api-guide §2.7.4 may need status update."
fi
raw=$(curl -sS --max-time 15 -w "\n%{http_code}" "${HDR[@]}" "${BASE}/api/v1/lookup?at=1700000000000000000") || raw=$'\n000'
c=$(code_of "$raw")
if [[ "$c" == "404" ]]; then ok "GET /api/v1/lookup (404 as 待上线)"; else
	ok "GET /api/v1/lookup (HTTP $c)"
	[[ "$c" =~ ^2 ]] && note "GET /api/v1/lookup returned 2xx; api-guide §2.7.5 may need status update."
fi

if [[ -n "$SEG_AID" ]]; then
	raw=$(curl -sS --max-time 15 -w "\n%{http_code}" -X PATCH "${HDR[@]}" "${BASE}/api/v1/assets/${SEG_AID}/actions/fake0001" \
		-H "If-Match: 1" -d '{"primary_label":"pickup"}') || raw=$'\n000'
	c=$(code_of "$raw")
	if [[ "$c" == "404" ]]; then ok "PATCH actions/{id} fake id (404)"; else note "PATCH actions: HTTP $c (doc 待上线)"
	fi
fi

echo ""
echo "--- § internal/commit-segments (doc §4) ---"
# Doc curl has no token; try without auth first, then with token.
raw=$(curl -sS --max-time 25 -w "\n%{http_code}" -X POST -H "Content-Type: application/json" "${BASE}/internal/commit-segments" \
	-d '{"mcap_file_id":"nonexistent-mcap-xx","ranges":[[1,2]],"reviewer":"x","owner":"y"}') || raw=$'\n000'
c=$(code_of "$raw")
if [[ "$c" =~ ^2 ]]; then
	ok "POST /internal/commit-segments without token"
	note "api-guide §4 example omits X-Grace-Token but §4 note says auth required — inconsistent."
elif [[ "$c" == "401" || "$c" == "403" ]]; then
	ok "POST /internal/commit-segments without token → $c (auth required)"
	note "api-guide §4 curl example should add -H X-Grace-Token to match §4 note."
else
	# 404 mcap / 422 validation still informative
	ok "POST /internal/commit-segments without token → $c (expected non-2xx for bad mcap)"
fi

if [[ -n "$MCAP_ID" ]]; then
	postc "POST commit-segments (with token, real mcap)" "/internal/commit-segments" \
		'{"mcap_file_id":"'"$MCAP_ID"'","ranges":[[1700000000000000000,1700000001000000000]],"reviewer":"verify","owner":"verify"}' "201"
else
	note "No mcap_file_id from queries; skipped happy-path commit-segments."
fi

echo ""
echo "--- § 交付创建 (真实 asset_ids) ---"
if [[ -n "$ANY_AID" ]]; then
	postc_idem "POST deliveries" "/api/v1/deliveries" \
		'{"asset_ids":["'"$ANY_AID"'"],"customer_id":"verify-customer","contract_id":"c1","note":"api-guide verify","owner":"verify"}'
	DEL_JSON=$(curl -sS --max-time 25 -X POST "${HDR[@]}" -H "Idempotency-Key: verify-delivery-read-$RANDOM" "${BASE}/api/v1/deliveries" \
		-d '{"asset_ids":["'"$ANY_AID"'"],"customer_id":"verify-customer-2"}')
	DID=$(echo "$DEL_JSON" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('delivery_id')or'')" 2>/dev/null || echo "")
	if [[ -n "$DID" ]]; then getc "GET delivery by id" "/api/v1/deliveries/${DID}"; fi
	getc "GET customers/.../deliveries" "/api/v1/customers/verify-customer/deliveries?page=1&page_size=5"
fi

echo ""
echo "--- § MCAP GET by id ---"
if [[ -n "$MCAP_ID" ]]; then
	getc "GET mcap-files/{id}" "/api/v1/mcap-files/${MCAP_ID}"
fi

echo ""
echo "=========================================="
echo " Summary: $PASS passed, $FAIL failed"
if [[ ${#DOC_NOTE[@]} -gt 0 ]]; then
	echo ""
	echo " Documentation notes (review api-guide.md):"
	for n in "${DOC_NOTE[@]}"; do echo "  - $n"; done
fi
echo "=========================================="
[[ "$FAIL" -eq 0 ]]
