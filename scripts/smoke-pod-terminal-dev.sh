#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

WORKFLOW_NAME="${WORKFLOW_NAME:-}"
NODE_ID="${NODE_ID:-}"

if [[ -z "$WORKFLOW_NAME" || -z "$NODE_ID" ]]; then
  echo "SKIP: set WORKFLOW_NAME and NODE_ID to smoke Pod terminal policy"
  exit 0
fi

source "$ROOT/scripts/dev-backend-env.sh"

tmp="$(mktemp)"
code="$(
  curl -sS -o "$tmp" -w "%{http_code}" \
    "${API_HDR[@]}" \
    -H "Content-Type: application/json" \
    -d '{"command":"sh"}' \
    "$BASE/api/v1/workflows/$WORKFLOW_NAME/nodes/$NODE_ID/terminal-sessions"
)"

if [[ "$code" == "403" ]] && grep -q "POD_EXEC_FORBIDDEN" "$tmp"; then
  echo "ok: default terminal policy is disabled"
  rm -f "$tmp"
  exit 0
fi

if [[ "$code" == "201" ]]; then
  session_id="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["id"])' "$tmp")"
  curl -sS "${API_HDR[@]}" -X POST "$BASE/api/v1/pod-terminal/sessions/$session_id/terminate" >/dev/null
  echo "ok: terminal session create/terminate succeeded"
  rm -f "$tmp"
  exit 0
fi

echo "unexpected status $code"
cat "$tmp"
rm -f "$tmp"
exit 1
