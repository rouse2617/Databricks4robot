#!/usr/bin/env bash
set -euo pipefail

CONNECT_URL="${CONNECT_URL:-http://localhost:8083}"

# Change to the directory containing this script so that relative paths work
cd "$(dirname "$0")"

register() {
  local file="$1"
  echo "Registering $(basename "$file")"
  curl -sS -X POST "${CONNECT_URL}/connectors" \
    -H 'Content-Type: application/json' \
    --data "@${file}"
  echo
}

register "connectors/postgres-asset-events.json"
register "connectors/postgres-current-state.json"
