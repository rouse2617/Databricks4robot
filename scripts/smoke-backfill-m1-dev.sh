#!/usr/bin/env bash
set -euo pipefail

if [[ -f "$(dirname "$0")/dev-backend-env.sh" ]]; then
  # shellcheck source=/dev/null
  source "$(dirname "$0")/dev-backend-env.sh"
fi

: "${BASE:?BASE is required, e.g. https://backend-dev.example.com}"
: "${DATABREW_TOKEN:?DATABREW_TOKEN is required}"
: "${BACKFILL_JOB_ID:?BACKFILL_JOB_ID is required}"

auth=(-H "X-Databrew-Token: ${DATABREW_TOKEN}")

curl -fsS "${BASE}/api/v1/backfill/${BACKFILL_JOB_ID}/node-summary" "${auth[@]}" \
  | jq -e '.batchJobId and (.nodes | type == "array")' >/dev/null

status_code="$(
  curl -sS -o /tmp/backfill-node-failures-missing-node.json -w '%{http_code}' \
    "${BASE}/api/v1/backfill/${BACKFILL_JOB_ID}/node-failures" "${auth[@]}"
)"
[[ "${status_code}" == "400" ]] || {
  echo "expected node-failures missing pipelineNodeId to return 400, got ${status_code}" >&2
  exit 1
}

status_code="$(
  curl -sS -o /tmp/backfill-rerun-invalid-node.json -w '%{http_code}' \
    -X POST "${BASE}/api/v1/backfill/${BACKFILL_JOB_ID}/rerun" \
    "${auth[@]}" -H "Content-Type: application/json" \
    -d '{"scope":"node_failed","dryRun":true}'
)"
[[ "${status_code}" == "400" ]] || {
  echo "expected rerun node_failed without pipelineNodeId to return 400, got ${status_code}" >&2
  exit 1
}

echo "backfill m1 smoke passed"
