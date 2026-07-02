#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if [[ -f "$ROOT/scripts/dev-backend-env.sh" ]]; then
  # shellcheck disable=SC1091
  source "$ROOT/scripts/dev-backend-env.sh"
fi

BASE="${BASE:-${DATABREW_BASE_URL:-http://localhost:8080}}"
TOKEN="${DATABREW_TOKEN:-${TOKEN:-dev-token}}"
WORKFLOW_NAME="${WORKFLOW_NAME:-}"
NODE_ID="${NODE_ID:-}"

request() {
  local path="$1"
  curl -sS -w "\n%{http_code}" \
    -H "X-Databrew-Token: $TOKEN" \
    "$BASE$path"
}

if [[ -z "$WORKFLOW_NAME" || -z "$NODE_ID" ]]; then
  echo "No WORKFLOW_NAME/NODE_ID provided; checking documented error path."
  out="$(request "/api/v1/workflows/__missing_workflow__/nodes/__missing_node__/pod")"
  status="$(printf '%s' "$out" | tail -n1)"
  body="$(printf '%s' "$out" | sed '$d')"
  if [[ "$status" != "404" && "$status" != "503" ]]; then
    echo "expected 404/503 for missing or unavailable diagnostics, got $status"
    echo "$body"
    exit 1
  fi
  echo "$body" | grep -Eq '"code":"(WORKFLOW_NOT_FOUND|K8S_UNAVAILABLE)"'
  echo "ok: pod diagnostics error path returned $status"
  exit 0
fi

out="$(request "/api/v1/workflows/${WORKFLOW_NAME}/nodes/${NODE_ID}/pod")"
status="$(printf '%s' "$out" | tail -n1)"
body="$(printf '%s' "$out" | sed '$d')"
if [[ "$status" != "200" ]]; then
  echo "expected 200, got $status"
  echo "$body"
  exit 1
fi
echo "$body" | grep -q '"podName"'
echo "$body" | grep -q '"containers"'
echo "ok: pod diagnostics happy path returned 200"
