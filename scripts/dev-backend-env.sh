#!/usr/bin/env bash
# Canonical dev backend API environment (Cloud Run — not the public Gateway hostname).
#
# Usage:
#   source scripts/dev-backend-env.sh
#   curl -sfS "${API_HDR[@]}" "${BASE}/api/v1/customers"
#
# Optional overrides: GCP_PROJECT, GCP_REGION, BACKEND_SERVICE, GRACE_TOKEN, BASE
set -euo pipefail

PROJECT="${GCP_PROJECT:-green-valley-442103}"
REGION="${GCP_REGION:-us-central1}"
SERVICE="${BACKEND_SERVICE:-cyber-databrew-backend-dev}"

if [[ -z "${BASE:-}" ]]; then
  BASE="$(gcloud run services describe "$SERVICE" \
    --region="$REGION" --project="$PROJECT" \
    --format='value(status.url)' 2>/dev/null || true)"
  if [[ -z "$BASE" ]]; then
    echo "dev-backend-env: failed to resolve Cloud Run URL for $SERVICE" >&2
    return 1 2>/dev/null || exit 1
  fi
fi
BASE="${BASE%/}"

if [[ -z "${GRACE_TOKEN:-}" ]]; then
  GRACE_TOKEN="$(gcloud run services describe "$SERVICE" \
    --region="$REGION" --project="$PROJECT" --format=json \
    | python3 -c "import sys,json; svc=json.load(sys.stdin); envs={e['name']:e.get('value','') for e in svc['spec']['template']['spec']['containers'][0].get('env',[])}; print(envs.get('GRACE_TOKEN',''))")"
fi

if [[ -z "${CLOUDRUN_ID_TOKEN:-}" ]]; then
  CLOUDRUN_ID_TOKEN="$(gcloud auth print-identity-token 2>/dev/null || true)"
fi

export BASE GRACE_TOKEN CLOUDRUN_ID_TOKEN
export TOKEN="${TOKEN:-$GRACE_TOKEN}"

API_HDR=( -H "X-Grace-Token: ${GRACE_TOKEN}" -H "Content-Type: application/json" )
if [[ -n "${CLOUDRUN_ID_TOKEN}" ]]; then
  API_HDR+=( -H "Authorization: Bearer ${CLOUDRUN_ID_TOKEN}" )
fi
export API_HDR

# shellcheck disable=SC2034
DEV_BACKEND_NOTE="Canonical dev API is Cloud Run (${SERVICE}). Do not use api-cyber-databrew-dev.cyberorigin.ai for deploy verification unless docs explicitly say Gateway was updated."
