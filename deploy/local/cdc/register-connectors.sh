#!/usr/bin/env bash
set -euo pipefail

CONNECT_URL="${CONNECT_URL:-http://localhost:8084}"

# Change to the directory containing this script so that relative paths work
cd "$(dirname "$0")"

register() {
  local file="$1"
  local name
  name="$(basename "$file" .json)"

  if curl -sf "${CONNECT_URL}/connectors/${name}" >/dev/null 2>&1; then
    echo "Connector ${name} already exists, skipping"
    return 0
  fi

  echo "Registering $(basename "$file")"
  curl -sS -X POST "${CONNECT_URL}/connectors" \
    -H 'Content-Type: application/json' \
    --data "@${file}"
  echo
}

register "connectors/postgres-unified-cdc.json"
