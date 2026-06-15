#!/usr/bin/env bash
set -euo pipefail

if [[ -f "$(dirname "$0")/dev-backend-env.sh" ]]; then
  # shellcheck source=/dev/null
  source "$(dirname "$0")/dev-backend-env.sh"
fi

: "${BASE:?BASE is required}"
: "${DATABREW_TOKEN:?DATABREW_TOKEN is required}"
: "${BACKFILL_RESULT_ASSET_ID:?BACKFILL_RESULT_ASSET_ID is required}"

auth=(-H "X-Databrew-Token: ${DATABREW_TOKEN}" -H "Content-Type: application/json")
report_id="${BACKFILL_RESULT_REPORT_ID:-report.project@1.0.0-backfill-left-eye-only}"
version="${BACKFILL_RESULT_VERSION:-1.0.0}"

curl -fsS -X POST "${BASE}/api/v1/backfill/results" "${auth[@]}" \
  -d "$(jq -nc \
    --arg assetId "${BACKFILL_RESULT_ASSET_ID}" \
    --arg reportId "${report_id}" \
    --arg version "${version}" \
    '{assetId:$assetId, reportId:$reportId, version:$version, manifest:{assetId:$assetId}, result:{status:"ok", source:"smoke"}}')" \
  | jq -e '.status == "registered" and .assetId' >/dev/null

status_code="$(
  curl -sS -o /tmp/backfill-results-mismatch.json -w '%{http_code}' \
    -X POST "${BASE}/api/v1/backfill/results" "${auth[@]}" \
    -d "$(jq -nc \
      --arg assetId "${BACKFILL_RESULT_ASSET_ID}" \
      --arg reportId "${report_id}" \
      --arg version "${version}" \
      '{assetId:$assetId, reportId:$reportId, version:$version, manifest:{assetId:"other-asset"}, result:{}}')"
)"
[[ "${status_code}" == "400" ]] || {
  echo "expected manifest mismatch to return 400, got ${status_code}" >&2
  exit 1
}

echo "backfill results api smoke passed"
