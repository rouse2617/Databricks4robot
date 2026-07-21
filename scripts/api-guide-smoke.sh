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
PIPELINE_CONFIG_SMOKE_ID=""

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

put_json() {
	local name="$1" path="$2" data="$3"
	local raw
	raw=$(curl -sS --max-time 30 -w "\n%{http_code}" -X PUT "${API_HDR[@]}" "$BASE$path" -d "$data" 2>/dev/null) || raw=$'\n000'
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

delete() {
	local name="$1" path="$2"
	local raw
	raw=$(curl -sS --max-time 30 -w "\n%{http_code}" -X DELETE "${API_HDR[@]}" "$BASE$path" 2>/dev/null) || raw=$'\n000'
	RESP_CODE=$(echo "$raw" | tail -n1)
	RESP_BODY=$(echo "$raw" | sed '$d')
	if [[ "$RESP_CODE" =~ ^2 ]]; then ok "$name"; else bad "$name"; fi
	echo "$RESP_BODY"
}

expect_code_delete() {
	local name="$1" path="$2" expected="$3"
	local raw code body
	raw=$(curl -sS --max-time 30 -w "\n%{http_code}" -X DELETE "${API_HDR[@]}" "$BASE$path" 2>/dev/null) || raw=$'\n000'
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
echo "--- § Pipeline execution targets ---"
get "execution-targets" "/api/v1/execution-targets"
if [[ "$RESP_CODE" == "200" ]]; then
	if echo "$RESP_BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); items=d.get('items', []); assert isinstance(items, list); assert all('id' in i and 'namespace' in i for i in items)" 2>/dev/null; then
		ok "execution-targets response shape"
	else
		bad "execution-targets response shape"
	fi
fi
get "pipeline runtime mounts" "/api/v1/pipeline/runtime-mounts"
if [[ "$RESP_CODE" == "200" ]]; then
	if echo "$RESP_BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); assert isinstance(d.get('secrets', []), list); assert isinstance(d.get('storage', []), list); assert all('id' in i and 'kind' in i and 'defaultMountPath' in i for i in d.get('storage', []))" 2>/dev/null; then
		ok "pipeline runtime mounts response shape"
	else
		bad "pipeline runtime mounts response shape"
	fi
fi
get "pipeline-runs" "/api/v1/pipeline-runs"
if [[ "$RESP_CODE" == "200" ]]; then
	if echo "$RESP_BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); items=d.get('items', []); assert isinstance(items, list); assert all('id' in i and 'status' in i and 'workflowName' in i for i in items)" 2>/dev/null; then
		ok "pipeline-runs response shape"
	else
		bad "pipeline-runs response shape"
	fi
fi
get "runs summary with search" "/api/v1/runs?view=summary&q=pipeline&page=1&pageSize=5"
if [[ "$RESP_CODE" == "200" ]]; then
	if echo "$RESP_BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); items=d.get('items', []); assert isinstance(items, list); assert all(('totalEstimatedCost' not in i) or (i.get('totalEstimatedCost') is None) or isinstance(i.get('totalEstimatedCost'), (int,float)) for i in items)" 2>/dev/null; then
		ok "runs summary search/cost response shape"
	else
		bad "runs summary search/cost response shape"
	fi
fi
if [[ -n "${PIPELINE_TEMPLATE_ID:-}" ]]; then
	inline_run_payload='{"target_id":"default","configSelection":{"mode":"inline","fileName":"runtime-config.yaml","content":"foo: bar\nnested:\n  enabled: true","mountPath":"/workspace/configs","targetFilename":"app-config.yaml"}}'
	inline_run_body=$(post_json "pipeline run template with inline configSelection" "/api/v1/pipeline-runs/template/${PIPELINE_TEMPLATE_ID}" "$inline_run_payload")
	if echo "$inline_run_body" | python3 -c 'import sys,json; d=json.load(sys.stdin); assert d.get("id"); assert d.get("executionTargetId") == "default"' 2>/dev/null; then
		ok "pipeline run template inline configSelection response shape"
	else
		RESP_CODE="json"
		RESP_BODY="$inline_run_body"
		bad "pipeline run template inline configSelection response shape"
	fi
	if [[ -n "$PIPELINE_CONFIG_SMOKE_ID" ]]; then
		saved_run_payload='{"target_id":"default","configSelection":{"mode":"saved","configId":"'"${PIPELINE_CONFIG_SMOKE_ID}"'","version":1,"fileName":"smoke-config.yaml","mountPath":"/workspace/configs","targetFilename":"saved-config.yaml"}}'
		saved_run_body=$(post_json "pipeline run template with saved configSelection" "/api/v1/pipeline-runs/template/${PIPELINE_TEMPLATE_ID}" "$saved_run_payload")
		if echo "$saved_run_body" | python3 -c 'import sys,json; d=json.load(sys.stdin); assert d.get("id"); assert d.get("executionTargetId") == "default"' 2>/dev/null; then
			ok "pipeline run template saved configSelection response shape"
		else
			RESP_CODE="json"
			RESP_BODY="$saved_run_body"
			bad "pipeline run template saved configSelection response shape"
		fi
	else
		echo "  skip pipeline run template saved configSelection — set RUN_WRITES=1 so smoke can create a disposable pipeline config first"
	fi
else
	echo "  skip pipeline run template configSelection smoke — set PIPELINE_TEMPLATE_ID to a disposable template id"
fi
expect_code_post "pipeline save duplicate fan-in -> 400" "/api/v1/pipelines" '{"name":"smoke-invalid-fanin","pipeline":{"name":"smoke-invalid-fanin","nodes":[{"id":"a","component":{"name":"a","image":"busybox","command":["sh","-c"],"args":[{"name":"script","value":"echo a > /tmp/outputs/output"}]},"outputs":[{"name":"output","type":"string"}]},{"id":"b","component":{"name":"b","image":"busybox","command":["sh","-c"],"args":[{"name":"script","value":"echo b > /tmp/outputs/output"}]},"outputs":[{"name":"output","type":"string"}]},{"id":"join","component":{"name":"join","image":"busybox"},"inputs":[{"name":"input","type":"string"}]}],"edges":[{"source":"a.output","target":"join.input"},{"source":"b.output","target":"join.input"}]}}}' "400" >/dev/null

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
get "asset type schema dataset" "/api/v1/asset-types/dataset/schema"
expect_code_get "asset type schema unknown -> 404" "/api/v1/asset-types/unknown/schema" "404" >/dev/null

echo ""
echo "--- § pipeline component registry ---"
get "pipeline-configs list" "/api/v1/pipeline-configs"
expect_code_post "pipeline-configs missing content -> 400" "/api/v1/pipeline-configs" '{"name":"smoke-missing-content.yaml","lifecycle":"ready"}' "400" >/dev/null
if [[ "${RUN_WRITES:-0}" == "1" ]]; then
	config_name="smoke-config-$(date +%s).yaml"
	config_body='{"name":"'"${config_name}"'","description":"api guide smoke config","tags":["smoke","pipeline"],"lifecycle":"ready","content":"threshold: 0.82\nwindow: 5\n","summary":"initial smoke version"}'
	config_created=$(post_json "pipeline-configs create" "/api/v1/pipeline-configs" "$config_body")
	config_id=$(echo "$config_created" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("id",""))' 2>/dev/null || true)
	if [[ -n "$config_id" ]]; then
		PIPELINE_CONFIG_SMOKE_ID="$config_id"
		get "pipeline-configs get" "/api/v1/pipeline-configs/${config_id}"
		get "pipeline-configs get v1 content" "/api/v1/pipeline-configs/${config_id}/versions/1"
		config_v2_body='{"status":"ready","content":"threshold: 0.90\nwindow: 5\n","summary":"raise threshold"}'
		config_v2=$(post_json "pipeline-configs create v2" "/api/v1/pipeline-configs/${config_id}/versions" "$config_v2_body")
		if echo "$config_v2" | python3 -c 'import sys,json; d=json.load(sys.stdin); assert d.get("version") == 2 and "content" not in d' 2>/dev/null; then
			ok "pipeline-configs create v2 response shape"
		else
			RESP_CODE="json"
			RESP_BODY="$config_v2"
			bad "pipeline-configs create v2 response shape"
		fi
		put_json "pipeline-configs update metadata" "/api/v1/pipeline-configs/${config_id}" '{"name":"'"${config_name}"'","description":"updated api guide smoke config","tags":["smoke","updated"],"fileType":"yaml","lifecycle":"ready"}' >/dev/null
		node_config_pipeline_name="smoke-node-config-$(date +%s)"
		node_config_pipeline_body='{"name":"'"${node_config_pipeline_name}"'","pipeline":{"name":"'"${node_config_pipeline_name}"'","nodes":[{"id":"configured-step","component":{"name":"configured-step","image":"busybox","command":["sh","-c"],"args":[{"name":"script","value":"echo ok"}]},"runtimeConfig":{"mode":"saved","configId":"'"${config_id}"'","version":2,"fileName":"'"${config_name}"'","mountPath":"/workspace/configs","targetFilename":"smoke-config.yaml"},"inputs":[],"outputs":[]}],"edges":[]}}'
		node_config_template=$(post_json "pipelines create with node runtimeConfig" "/api/v1/pipelines" "$node_config_pipeline_body")
		if echo "$node_config_template" | python3 -c 'import sys,json; d=json.load(sys.stdin); rc=d.get("pipeline",{}).get("nodes",[{}])[0].get("runtimeConfig",{}); assert d.get("id") and rc.get("configId")' 2>/dev/null; then
			ok "pipelines create node runtimeConfig response shape"
		else
			RESP_CODE="json"
			RESP_BODY="$node_config_template"
			bad "pipelines create node runtimeConfig response shape"
		fi
		post_json "pipeline-configs deprecate" "/api/v1/pipeline-configs/${config_id}/deprecate" '{}' >/dev/null
	else
		RESP_CODE="json"
		RESP_BODY="$config_created"
		bad "pipeline-configs create id extraction"
	fi
else
	echo "  skip pipeline config write smoke — set RUN_WRITES=1 to create/update/deprecate disposable config records"
fi

get "pipeline-components list" "/api/v1/pipeline-components"
get "pipeline-component-releases list selectable" "/api/v1/pipeline-component-releases?selectable=true"
expect_code_get "pipeline-component-releases invalid selectable -> 400" "/api/v1/pipeline-component-releases?selectable=maybe" "400" >/dev/null
expect_code_post "pipeline-components missing image -> 400" "/api/v1/pipeline-components" '{"name":"smoke-missing-image","type":"container"}' "400" >/dev/null
if [[ "${RUN_WRITES:-0}" == "1" ]]; then
	component_name="smoke-component-$(date +%s)"
	component_body='{"name":"'"${component_name}"'","type":"container","description":"api guide smoke","image":"busybox","tag":"latest","command":["sh","-c"],"args":["echo ok"],"env":{"MODE":"smoke"}}'
	component_created=$(post_json "pipeline-components create" "/api/v1/pipeline-components" "$component_body")
	component_id=$(echo "$component_created" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("id",""))' 2>/dev/null || true)
	if [[ -n "$component_id" ]]; then
		get "pipeline-components get" "/api/v1/pipeline-components/${component_id}"
		put_json "pipeline-components update" "/api/v1/pipeline-components/${component_id}" '{"name":"'"${component_name}"'-updated","type":"container","image":"busybox","tag":"1.36"}' >/dev/null
		delete "pipeline-components delete" "/api/v1/pipeline-components/${component_id}" >/dev/null
	else
		RESP_CODE="json"
		RESP_BODY="$component_created"
		bad "pipeline-components create id extraction"
	fi
	release_label="main-smoke$(date +%s)"
	release_commit="abcdef$(date +%s)"
	release_body='{"source":{"provider":"api-guide-smoke","repo":"CyberOrigin2077/automated-processing-gcloud","ref":"refs/heads/main","refType":"branch","commit":"'"${release_commit}"'","buildId":"smoke-build","trigger":"smoke-trigger"},"items":[{"componentId":"smoke-task","taskName":"smoke-task","taskPath":"tasks/smoke_task","releaseLabel":"'"${release_label}"'","runtimeImage":"registry.example.com/smoke-task@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","runtimeSnapshot":{"command":["python","src/main.py"],"inputPorts":[{"name":"input","type":"asset"}],"outputPorts":[{"name":"output","type":"asset"}],"resources":{"cpu":"1","memory":"1Gi"}}}]}'
	release_created=$(post_json "pipeline-component-releases sync" "/api/v1/pipeline-component-releases/sync" "$release_body")
	release_id=$(echo "$release_created" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("items",[{}])[0].get("id",""))' 2>/dev/null || true)
	if [[ -n "$release_id" ]]; then
		get "pipeline-component-releases get" "/api/v1/pipeline-component-releases/${release_id}"
		get "pipeline-component-releases search source commit" "/api/v1/pipeline-component-releases?q=${release_commit}"
	else
		RESP_CODE="json"
		RESP_BODY="$release_created"
		bad "pipeline-component-releases sync id extraction"
	fi
	release_unpinned='{"items":[{"componentId":"smoke-unpinned","releaseLabel":"pr-1-abc123","runtimeImage":"registry.example.com/smoke-unpinned:abc123","runtimeSnapshot":{"command":["python","main.py"],"resources":{"cpu":"1"}}}]}'
	unpinned_resp=$(post_json "pipeline-component-releases missing digest -> unselectable" "/api/v1/pipeline-component-releases/sync" "$release_unpinned")
	if echo "$unpinned_resp" | python3 -c 'import sys,json; d=json.load(sys.stdin); item=d.get("items",[{}])[0]; assert item.get("selectable") is False and item.get("validationStatus") == "failed"' 2>/dev/null; then
		ok "pipeline-component-releases missing digest validation"
	else
		RESP_CODE="json"
		RESP_BODY="$unpinned_resp"
		bad "pipeline-component-releases missing digest validation"
	fi
else
	echo "  skip pipeline component write smoke — set RUN_WRITES=1 to create/update/delete disposable component records"
fi

echo ""
echo "--- § 资产 / 搜索 / 交付 / mcap-files ---"
get "search sync status" "/api/v1/search/sync-status"
expect_code_get "search lineage invalid direction -> 400" "/api/v1/search/assets?lineage_with=asset-smoke&lineage_direction=sideways" "400" >/dev/null
post "queries run (structured)" "/api/v1/queries/run" '{"schema_version":"v1","mode":"structured","scope":{"resource":"assets"},"page":{"page":1,"page_size":5}}' >/dev/null
post "queries run (include_history)" "/api/v1/queries/run?include_history=true" '{"schema_version":"v1","scope":{"resource":"assets","include_history":true},"page":{"page":1,"page_size":5}}' >/dev/null
post "queries run (keyword)" "/api/v1/queries/run" '{"schema_version":"v1","mode":"keyword","scope":{"resource":"assets"},"where":{"pred":{"field":"_fulltext","op":"ilike","value":"warehouse"}},"page":{"page":1,"page_size":5}}' >/dev/null

# CYB-3713 regression pack: keyword mode `q` param must be honored (was
# silently dropped, returning full unfiltered list). Compare filtered vs
# unfiltered totals to prove the injection is live.
KW_BODY=$(curl -sS --max-time 20 -X POST "${API_HDR[@]}" "${BASE}/api/v1/queries/run" \
	-d '{"schema_version":"v1","mode":"structured","scope":{"resource":"assets"},"page":{"page":1,"page_size":1}}' 2>/dev/null || echo '{}')
KW_TOTAL_UNFILTERED=$(echo "$KW_BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('total', -1))" 2>/dev/null || echo -1)

for KWQ in "nonexistent-xxx-cyb-3713-regression-do-not-match" "备餐操作"; do
	RAW=$(curl -sS --max-time 20 -w "\n%{http_code}" -X POST "${API_HDR[@]}" "${BASE}/api/v1/queries/run" \
		-d "{\"schema_version\":\"v1\",\"mode\":\"keyword\",\"q\":\"${KWQ}\",\"scope\":{\"resource\":\"assets\"},\"page\":{\"page\":1,\"page_size\":1}}" 2>/dev/null || echo $'\n000')
	RESP_CODE=$(echo "$RAW" | tail -n1)
	RESP_BODY=$(echo "$RAW" | sed '$d')
	TOTAL=$(echo "$RESP_BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('total', -1))" 2>/dev/null || echo -1)
	if [[ "$RESP_CODE" != "200" ]]; then
		bad "queries run keyword q=${KWQ}"
	elif [[ "$TOTAL" == "$KW_TOTAL_UNFILTERED" ]]; then
		# Total == unfiltered means q was ignored — the pre-fix regression.
		FAIL=$((FAIL + 1))
		echo "  FAIL queries run keyword q=${KWQ}: total ${TOTAL} equals unfiltered ${KW_TOTAL_UNFILTERED} (q silently dropped — CYB-3713 regression)"
	else
		ok "queries run keyword q=${KWQ} (total=${TOTAL} != unfiltered ${KW_TOTAL_UNFILTERED})"
	fi
done
get "deliveries list" "/api/v1/deliveries?page=1&page_size=5"
get "mcap-files list" "/api/v1/mcap-files?page=1&page_size=5"

echo ""
echo "--- § workflow monitoring / operations ---"
get "workflows list" "/api/v1/workflows"
if [[ -n "${WORKFLOW_NAME:-}" ]]; then
	get "workflow detail" "/api/v1/workflows/${WORKFLOW_NAME}"
	if echo "$RESP_BODY" | python3 -c 'import sys,json; data=json.load(sys.stdin); assert isinstance(data.get("edges"), list)' 2>/dev/null; then
		ok "workflow detail edges contract"
	else
		bad "workflow detail edges contract"
	fi
	expect_code_get "workflow logs missing nodeId -> 400" "/api/v1/workflows/${WORKFLOW_NAME}/logs" "400" >/dev/null
	if [[ -n "${WORKFLOW_NODE_ID:-}" ]]; then
		get "workflow node logs bounded" "/api/v1/workflows/${WORKFLOW_NAME}/logs?nodeId=${WORKFLOW_NODE_ID}&tailLines=50&limitBytes=65536"
		if echo "$RESP_BODY" | python3 -c 'import sys,json; data=json.load(sys.stdin); assert data.get("truncation", {}).get("bounded") is True; assert data.get("pagination", {}).get("available") is False; assert data.get("window", {}).get("scope") == "bounded-live-window"; assert "lineCount" in data' 2>/dev/null; then
			ok "workflow logs bounded metadata"
		else
			bad "workflow logs bounded metadata"
		fi
		expect_code_get "workflow logs cursor unavailable -> 400" "/api/v1/workflows/${WORKFLOW_NAME}/logs?nodeId=${WORKFLOW_NODE_ID}&cursor=older" "400" >/dev/null
		raw=$(curl -sS -N --max-time 3 -D - -o /dev/null "${API_HDR[@]}" "${BASE}/api/v1/workflows/${WORKFLOW_NAME}/logs/stream?nodeId=${WORKFLOW_NODE_ID}&tailLines=1&limitBytes=4096" 2>/dev/null || true)
		if echo "$raw" | grep -qi "Content-Type: text/event-stream"; then
			ok "workflow logs stream content-type"
		else
			bad "workflow logs stream content-type"
		fi
	else
		echo "  skip workflow logs happy path — set WORKFLOW_NODE_ID to exercise GET /workflows/{name}/logs"
	fi
else
	echo "  skip workflow detail/log smoke — set WORKFLOW_NAME to exercise GET /workflows/{name}"
fi
expect_code_get "workflow missing detail -> 404" "/api/v1/workflows/__missing_workflow__" "404" >/dev/null
for op in retry resubmit suspend resume terminate; do
	expect_code_post "workflow ${op} missing workflow -> 500" "/api/v1/workflows/__missing_workflow__/${op}" "{}" "500" >/dev/null
done
if [[ -n "${WORKFLOW_OPERATION_NAME:-}" ]]; then
	for op in retry resubmit suspend resume terminate; do
		post "workflow ${op}" "/api/v1/workflows/${WORKFLOW_OPERATION_NAME}/${op}" "{}" >/dev/null
	done
else
	echo "  skip workflow operation happy paths — set WORKFLOW_OPERATION_NAME to a disposable workflow"
fi
expect_code_delete "workflow delete missing workflow -> 500" "/api/v1/workflows/__missing_workflow__" "500" >/dev/null
if [[ -n "${WORKFLOW_DELETE_NAME:-}" ]]; then
	delete "workflow delete" "/api/v1/workflows/${WORKFLOW_DELETE_NAME}" >/dev/null
else
	echo "  skip workflow delete happy path — set WORKFLOW_DELETE_NAME to a disposable workflow"
fi

echo ""
echo "--- § algo-runs (CYB-1018/CYB-1123) ---"
get "GET algo-runs list (page/page_size)" "/api/v1/algo-runs?page=1&page_size=5"
if [[ "${RUN_WRITES:-0}" == "1" ]]; then
	RUN_ID=$(python3 -c "import secrets,string; a=string.ascii_letters+string.digits; print(''.join(secrets.choice(a) for _ in range(16)))")
	post "POST algo-runs" "/api/v1/algo-runs" "{\"run_id\":\"${RUN_ID}\",\"algo_name\":\"hand_track\",\"algo_version\":\"2.0\",\"algo_kind\":\"processing\",\"triggered_by\":\"manual:api-guide-smoke\"}" >/dev/null
	expect_code_post "POST algo-runs duplicate run_id -> 409" "/api/v1/algo-runs" "{\"run_id\":\"${RUN_ID}\",\"algo_name\":\"hand_track\",\"algo_version\":\"2.0\",\"algo_kind\":\"processing\",\"triggered_by\":\"manual:api-guide-smoke\"}" "409" >/dev/null
	MISSING_ASSET_RUN_ID=$(python3 -c "import secrets,string; a=string.ascii_letters+string.digits; print(''.join(secrets.choice(a) for _ in range(16)))")
	expect_code_post "POST algo-runs missing input asset -> 400" "/api/v1/algo-runs" "{\"run_id\":\"${MISSING_ASSET_RUN_ID}\",\"algo_name\":\"hand_track\",\"algo_version\":\"2.0\",\"algo_kind\":\"processing\",\"triggered_by\":\"manual:api-guide-smoke\",\"input_asset_ids\":[\"DEAD1536\"]}" "400" >/dev/null
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

# ── Layered child-asset creation (CYB-1222) ──────────────────────────────
echo "=== 1.10 Layered child-asset creation ==="

# Error path: non-existent parent
expect_code_post "POST /assets/{id}/clips non-existent parent → 404" "/api/v1/assets/zzzzzzzz/clips" \
	'{"start_timestamp_ns":1700000000000000000,"end_timestamp_ns":1700000000100000000,"split_method":"manual"}' "404"
expect_code_post "POST /assets/{id}/tasks non-existent parent → 404" "/api/v1/assets/zzzzzzzz/tasks" \
	'{"start_timestamp_ns":1700000000000000000,"end_timestamp_ns":1700000000100000000,"split_method":"manual"}' "404"
expect_code_post "POST /assets/{id}/frames non-existent parent → 404" "/api/v1/assets/zzzzzzzz/frames" \
	'{"start_timestamp_ns":1700000000000000000,"end_timestamp_ns":1700000000100000000,"split_method":"manual"}' "404"

if [[ "${RUN_WRITES:-0}" == "1" ]]; then
	# Look up a segment to use as parent.
	SEG_ID=$(echo '{"schema_version":"v1","scope":{"resource":"assets"},"select":{"fields":["asset_id"]},"where":{"and":[{"pred":{"field":"asset_type","op":"eq","value":"segment"}}]},"sort":[{"field":"created_at","direction":"desc"}],"page":{"page":1,"page_size":1}}' \
		| post_json "layered find segment parent" "/api/v1/queries/run" | python3 -c "import sys,json; items=json.load(sys.stdin).get('items',[]); print(items[0]['asset_id'] if items else '')" 2>/dev/null || true)

	if [[ -n "$SEG_ID" ]]; then
		post "POST /assets/{id}/clips (happy)" "/api/v1/assets/${SEG_ID}/clips" \
			'{"start_timestamp_ns":1700000000000000000,"end_timestamp_ns":1700000000100000000,"split_method":"manual"}' >/dev/null
		post "POST /assets/{id}/frames (happy)" "/api/v1/assets/${SEG_ID}/frames" \
			'{"start_timestamp_ns":1700000000000000000,"end_timestamp_ns":1700000000100000000,"split_method":"manual"}' >/dev/null
		post "POST /assets/{id}/tasks (happy)" "/api/v1/assets/${SEG_ID}/tasks" \
			'{"start_timestamp_ns":1700000000000000000,"end_timestamp_ns":1700000000100000000,"split_method":"algo:hand_track@2.0","split_run_id":"R001"}' >/dev/null
	else
		echo "  WARN layered happy-path skipped: no segment found"
	fi
else
	echo "  skip layered write smoke — set RUN_WRITES=1 to exercise POST /assets/{id}/{clips,frames,tasks}"
fi

echo ""
echo "=== done: ${PASS} passed, ${FAIL} failed ==="
[[ "$FAIL" -eq 0 ]]
