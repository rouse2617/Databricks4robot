#!/usr/bin/env bash
# Business scenario: (1) asset tag upsert (2) query by tag (3) deliver + query deliveries.
# Aligns with docs/review/api-guide.md §2.4, §1.3, §3.x and backend/config/tag_registry.yaml.
#
# Usage (from repo root):
#   Default BASE is hosted **dev** backend (Cloud Run); see deploy/cloudrun/*.sh.
#   TOKEN=dev-token bash scripts/scenario-tags-delivery.sh
#   Local override: BASE=http://127.0.0.1:8080 TOKEN=dev-token bash scripts/scenario-tags-delivery.sh
#   If run.app URL drifts: gcloud run services describe cyber-databrew-backend-dev --region=us-central1 --format='value(status.url)'
#
# Optional:
#   ES_TAG_WAIT_SEC=3   sleep before tag-based queries/run (outbox → ES lag in dev)
#   CLEANUP=1           DELETE the created asset at the end
set -euo pipefail

BASE="${BASE:-https://cyber-databrew-backend-dev-wtttm6suaq-uc.a.run.app}"
TOKEN="${TOKEN:-dev-token}"
ES_TAG_WAIT_SEC="${ES_TAG_WAIT_SEC:-3}"
CLEANUP="${CLEANUP:-0}"

TS="$(date +%s)"
RUN_ID="tagdlv-${TS}"
OWNER="scenario-owner-${TS}"
CUSTOMER_ID="tag-scenario-cust-${TS}"
BATCH_TAG="scenario-batch-${TS}"

hdr=( -H "X-Databrew-Token: ${TOKEN}" -H "Content-Type: application/json" )

die() { echo "ERROR: $*" >&2; exit 1; }

json_post() {
	local path="$1" data="$2"
	curl -sS --max-time 60 -w "\n%{http_code}" -X POST "${hdr[@]}" "${BASE}${path}" -d "${data}" 2>/dev/null || echo -e "\n000"
}

json_get() {
	local path="$1"
	curl -sS --max-time 60 -w "\n%{http_code}" "${hdr[@]}" -H "Content-Type: application/json" "${BASE}${path}" 2>/dev/null || echo -e "\n000"
}

parse_http_body_code() {
	local raw="$1"
	export LAST_HTTP_BODY LAST_HTTP_CODE
	LAST_HTTP_CODE=$(echo "$raw" | tail -n1)
	LAST_HTTP_BODY=$(echo "$raw" | sed '$d')
}

expect_2xx() {
	local label="$1"
	[[ "$LAST_HTTP_CODE" =~ ^2 ]] || die "${label}: HTTP ${LAST_HTTP_CODE} body=${LAST_HTTP_BODY:0:400}"
}

echo "=== scenario: tags → query by tag → delivery → delivery queries === BASE=${BASE}"

# --- Seed: mcap-file + asset (unique raw_hash + ids) ---
MCAP_ID=$(python3 -c "import secrets,string; a=string.ascii_letters+string.digits; print(''.join(secrets.choice(a) for _ in range(8)))")
HASH=$(openssl rand -hex 16 2>/dev/null || python3 -c "import secrets; print(secrets.token_hex(16))")

MCAP_JSON=$(cat <<EOF
{
  "mcap_file_id": "${MCAP_ID}",
  "gcs_path": "gs://scenario-tags-delivery/path/${RUN_ID}.mcap",
  "raw_hash_md5": "${HASH}",
  "ingest_state": "summarized",
  "size_bytes": 1048576,
  "file_duration_ms": 60000,
  "start_timestamp_ns": 1700000000000000000,
  "end_timestamp_ns": 1700000060000000000,
  "channel_count": 12,
  "chunk_count": 5,
  "vendor_id": "scenario",
  "device_id": "scenario-1",
  "scene_id": "indoor",
  "owner": "${OWNER}"
}
EOF
)

echo ""
echo "--- (0) POST /api/v1/mcap-files ---"
raw=$(json_post "/api/v1/mcap-files" "${MCAP_JSON}")
parse_http_body_code "$raw"
expect_2xx "POST mcap-files"

ASSET_JSON=$(cat <<EOF
{
  "mcap_file_id": "${MCAP_ID}",
  "start_timestamp_ns": 1700000000000000000,
  "end_timestamp_ns": 1700000060000000000,
  "reviewer": "scenario",
  "owner": "${OWNER}",
  "type": "task_demo",
  "tags": { "priority": "low", "quality": "good", "scene": "indoor" }
}
EOF
)

echo "--- (0) POST /api/v1/assets ---"
raw=$(json_post "/api/v1/assets" "${ASSET_JSON}")
parse_http_body_code "$raw"
expect_2xx "POST assets"

ASSET_ID=$(echo "$LAST_HTTP_BODY" | python3 -c 'import sys,json; print(json.load(sys.stdin)["asset_id"])')
[[ -n "${ASSET_ID}" ]] || die "could not parse asset_id"
echo "    asset_id=${ASSET_ID}"

# --- (1) 资产打 tag：POST /api/v1/assets/{id}/tags ---
TAG_BODY=$(BATCH_TAG="${BATCH_TAG}" python3 <<'PY'
import json, os
print(json.dumps({"key": "batch", "value": os.environ["BATCH_TAG"]}))
PY
)

echo ""
echo "--- (1) POST /api/v1/assets/${ASSET_ID}/tags (registry key=batch) ---"
raw=$(json_post "/api/v1/assets/${ASSET_ID}/tags" "${TAG_BODY}")
parse_http_body_code "$raw"
expect_2xx "POST tags"
echo "    OK (tag upsert)"

# --- (2) 通过 tag 查询：POST /api/v1/queries/run ---
if [[ "${ES_TAG_WAIT_SEC}" =~ ^[0-9]+$ ]] && [[ "${ES_TAG_WAIT_SEC}" -gt 0 ]]; then
	echo "    waiting ${ES_TAG_WAIT_SEC}s for search index (dev ES/outbox lag)…"
	sleep "${ES_TAG_WAIT_SEC}"
fi

QUERY_JSON=$(OWNER="${OWNER}" BATCH_TAG="${BATCH_TAG}" python3 <<'PY'
import json, os

q = {
    "schema_version": "v1",
    "mode": "structured",
    "scope": {"resource": "assets"},
    "select": {"fields": ["asset_id", "owner", "tags", "lifecycle_state"]},
    "where": {
        "and": [
            {"pred": {"field": "owner", "op": "eq", "value": os.environ["OWNER"]}},
            {"pred": {"field": "tags_flat.batch", "op": "eq", "value": os.environ["BATCH_TAG"]}},
        ]
    },
    "page": {"page": 1, "page_size": 20},
}
print(json.dumps(q))
PY
)

echo ""
echo "--- (2) POST /api/v1/queries/run (owner + tags_flat.batch) ---"
raw=$(json_post "/api/v1/queries/run" "${QUERY_JSON}")
parse_http_body_code "$raw"
expect_2xx "queries/run"
FOUND=$(echo "$LAST_HTTP_BODY" | python3 -c 'import sys,json; d=json.load(sys.stdin); print(len(d.get("items") or []))')
echo "    items=${FOUND}"
[[ "${FOUND}" -ge 1 ]] || die "queries/run expected ≥1 row for this scenario"
FIRST=$(echo "$LAST_HTTP_BODY" | python3 -c 'import sys,json; d=json.load(sys.stdin); print(d["items"][0]["asset_id"])')
[[ "${FIRST}" == "${ASSET_ID}" ]] || die "first hit asset_id=${FIRST} want ${ASSET_ID}"

# --- (3) 交付 + 交付查询 ---
IDEM_KEY="scenario-delivery-${RUN_ID}"
DELIVERY_BODY=$(
	ASSET_ID="${ASSET_ID}" CUSTOMER_ID="${CUSTOMER_ID}" TS="${TS}" OWNER="${OWNER}" python3 <<'PY'
import json, os

print(
    json.dumps(
        {
            "asset_ids": [os.environ["ASSET_ID"]],
            "customer_id": os.environ["CUSTOMER_ID"],
            "contract_id": "contract-" + os.environ["TS"],
            "note": "scenario tag→query→delivery",
            "owner": os.environ["OWNER"],
        }
    )
)
PY
)

echo ""
echo "--- (3a) POST /api/v1/deliveries (Idempotency-Key) ---"
raw=$(curl -sS --max-time 60 -w "\n%{http_code}" -X POST \
	-H "X-Databrew-Token: ${TOKEN}" -H "Content-Type: application/json" \
	-H "Idempotency-Key: ${IDEM_KEY}" \
	"${BASE}/api/v1/deliveries" -d "${DELIVERY_BODY}" 2>/dev/null || echo -e "\n000")
parse_http_body_code "$raw"
expect_2xx "POST deliveries"
DELIVERY_ID=$(echo "$LAST_HTTP_BODY" | python3 -c 'import sys,json; print(json.load(sys.stdin)["delivery_id"])')
[[ -n "${DELIVERY_ID}" ]] || die "could not parse delivery_id"
echo "    delivery_id=${DELIVERY_ID}"

echo ""
echo "--- (3b) GET /api/v1/deliveries/${DELIVERY_ID} ---"
raw=$(json_get "/api/v1/deliveries/${DELIVERY_ID}")
parse_http_body_code "$raw"
expect_2xx "GET delivery by id"
echo "    status=$(echo "$LAST_HTTP_BODY" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("status",""))')"

echo ""
echo "--- (3c) GET /api/v1/customers/${CUSTOMER_ID}/deliveries?page=1&page_size=20 ---"
raw=$(json_get "/api/v1/customers/${CUSTOMER_ID}/deliveries?page=1&page_size=20")
parse_http_body_code "$raw"
expect_2xx "GET customer deliveries"
CNT=$(echo "$LAST_HTTP_BODY" | python3 -c 'import sys,json; d=json.load(sys.stdin); print(len(d.get("delivery_ids") or []))')
echo "    delivery_ids count=${CNT}"

echo ""
echo "--- (3d) GET /api/v1/deliveries?status=delivered&page=1&page_size=20 (list smoke) ---"
raw=$(json_get "/api/v1/deliveries?status=delivered&page=1&page_size=20")
parse_http_body_code "$raw"
expect_2xx "GET deliveries list"
LIST_TOTAL=$(echo "$LAST_HTTP_BODY" | python3 -c 'import sys,json; d=json.load(sys.stdin); print(d.get("total", "?"))')
echo "    list total (approx)=${LIST_TOTAL}"
IN_LIST=$(
	echo "$LAST_HTTP_BODY" | DELIVERY_ID="${DELIVERY_ID}" python3 -c 'import sys,json,os;d=json.load(sys.stdin);want=os.environ["DELIVERY_ID"];ids=[x.get("delivery_id") for x in (d.get("items") or [])];print(str(want in ids))'
)
if [[ "${IN_LIST}" == "True" ]]; then
	echo "    delivery_id appears on first page of delivered list"
else
	echo "    NOTE: delivery_id not on first page (expected if many deliveries); 3b+3c already verified this delivery"
fi

if [[ "${CLEANUP}" == "1" ]]; then
	echo ""
	echo "--- CLEANUP=1 DELETE /api/v1/assets/${ASSET_ID} ---"
	raw=$(curl -sS --max-time 30 -w "\n%{http_code}" -X DELETE "${hdr[@]}" "${BASE}/api/v1/assets/${ASSET_ID}" 2>/dev/null || echo -e "\n000")
	parse_http_body_code "$raw"
	expect_2xx "DELETE asset"
fi

echo ""
echo "=== scenario done ==="
