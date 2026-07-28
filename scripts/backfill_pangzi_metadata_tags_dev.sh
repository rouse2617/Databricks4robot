#!/usr/bin/env bash
# CYB-3714: Backfill task / source / city tags on pangzi-consumer mcap from
# their JSONB metadata (vibecap_tasks[], source_platform, location.address).
#
# Idempotent — POST /assets/:id/tags is Upsert; re-running the script against
# an already-populated set is a no-op at the DB level.
#
# Usage:
#   TOKEN="dbk_..." BASE=https://cyber-databrew-dev.cyberorigin.ai \
#     bash scripts/backfill_pangzi_metadata_tags_dev.sh [--dry-run]
#
# Requires: curl, jq, python3.
set -euo pipefail

BASE="${BASE:-https://cyber-databrew-dev.cyberorigin.ai}"
TOKEN="${TOKEN:?TOKEN env var required (dbk_ Bearer key with tags:write scope)}"
DRY_RUN=0
[[ "${1:-}" == "--dry-run" ]] && DRY_RUN=1

HDR=(-H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json")
OWNER="pangzi-consumer"
SOURCE_TYPE="system"
SOURCE_NAME="backfill_cyb_3714"

ok=0
skipped=0
failed=0

# Fetch all pangzi mcap. page_size big enough to cover current volume.
list_body=$(curl -sS --max-time 60 "${HDR[@]}" "$BASE/api/v1/mcap-files?owner=${OWNER}&page=1&page_size=500")
total=$(echo "$list_body" | jq -r '.total // 0')
echo "== ${OWNER} mcap total: ${total} =="

echo "$list_body" | jq -c '.items[] | {id: .mcap_file_id, tasks: (.metadata.vibecap_tasks // []), src: (.metadata.source_platform // ""), addr: (.metadata.location.address // "")}' | while IFS= read -r line; do
	id=$(echo "$line" | jq -r '.id')
	src=$(echo "$line" | jq -r '.src')
	addr=$(echo "$line" | jq -r '.addr')
	# Extract city — first comma-separated segment ending in 市.
	city=$(echo "$addr" | python3 -c "
import sys
addr = sys.stdin.read().strip()
for seg in addr.split(','):
	seg = seg.strip()
	if seg.endswith('市'):
		print(seg)
		break
")
	tasks_json=$(echo "$line" | jq -c '.tasks')
	tasks_count=$(echo "$line" | jq '.tasks | length')

	post_tag() {
		local key="$1" val="$2"
		[[ -z "$val" ]] && { skipped=$((skipped + 1)); return; }
		local payload
		payload=$(jq -cn --arg k "$key" --arg v "$val" --arg st "$SOURCE_TYPE" --arg sn "$SOURCE_NAME" \
			'{key:$k, value:$v, source_type:$st, source_name:$sn}')
		if [[ $DRY_RUN -eq 1 ]]; then
			echo "    DRY: POST /assets/${id}/tags $payload"
			ok=$((ok + 1))
			return
		fi
		local code
		code=$(curl -sS -o /dev/null -w "%{http_code}" --max-time 30 -X POST "${HDR[@]}" "$BASE/api/v1/assets/${id}/tags" -d "$payload")
		if [[ "$code" == "200" ]]; then
			ok=$((ok + 1))
		else
			failed=$((failed + 1))
			echo "    FAIL POST /tags key=${key} value=${val} asset=${id} → HTTP ${code}"
		fi
	}

	# 1) task — one row per vibecap_tasks entry
	if [[ "$tasks_count" -gt 0 ]]; then
		echo "$tasks_json" | jq -r '.[]' | while IFS= read -r t; do
			[[ -z "$t" || "$t" == "null" ]] && continue
			post_tag "task" "$t"
		done
	fi

	# 2) source — single row
	if [[ -n "$src" && "$src" != "null" ]]; then
		post_tag "source" "$src"
	fi

	# 3) city — best-effort parse
	if [[ -n "$city" ]]; then
		post_tag "city" "$city"
	fi

	echo "mcap=${id} tasks=${tasks_count} source=${src:-none} city=${city:-none}"
done

echo ""
echo "== backfill summary =="
echo "  ok:      ${ok}"
echo "  skipped: ${skipped}"
echo "  failed:  ${failed}"
if [[ "$failed" -gt 0 ]]; then
	exit 1
fi
