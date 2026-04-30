#!/usr/bin/env bash
set -euo pipefail

CONNECT_URL="${CONNECT_URL:-http://localhost:8084}"
ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"

register() {
  local file="$1"
  echo "Registering $(basename "$file")"
  curl -sS -X POST "${CONNECT_URL}/connectors" \
    -H 'Content-Type: application/json' \
    --data @"$file"
  echo
}

register "${ROOT_DIR}/connectors/postgres-asset-events.json"
register "${ROOT_DIR}/connectors/postgres-current-state.json"
