#!/usr/bin/env bash
# Legacy backfill result repro harness (CYB-2041 and related).
# Usage: bash scripts/test2-backfill-results-legacy.sh <case>
# Cases: right_eye | left_eye | both_eyes

set -euo pipefail

CASE="${1:-}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

usage() {
  echo "Usage: $0 <right_eye|left_eye|both_eyes>" >&2
  exit 1
}

require_env() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    echo "ERROR: ${name} is required" >&2
    exit 1
  fi
}

run_sql() {
  local sql_file="$1"
  if [[ ! -f "${sql_file}" ]]; then
    echo "ERROR: SQL file not found: ${sql_file}" >&2
    exit 1
  fi
  psql "${LEGACY_BACKFILL_DB_URL}" -v ON_ERROR_STOP=1 -f "${sql_file}"
}

case "${CASE}" in
  right_eye)
    require_env DATABREW_BASE_URL
    require_env DATABREW_TOKEN
    require_env LEGACY_BACKFILL_DB_URL
    require_env LEGACY_BACKFILL_ANCHOR_ASSET_ID
    run_sql "${REPO_ROOT}/backend/scripts/legacy-backfill-right-eye-only.sql"
    ;;
  left_eye)
    require_env DATABREW_BASE_URL
    require_env DATABREW_TOKEN
    require_env LEGACY_BACKFILL_DB_URL
    require_env LEGACY_BACKFILL_ANCHOR_ASSET_ID
    run_sql "${REPO_ROOT}/backend/scripts/legacy-backfill-left-eye-only.sql"
    ;;
  both_eyes)
    require_env DATABREW_BASE_URL
    require_env DATABREW_TOKEN
    require_env LEGACY_BACKFILL_DB_URL
    require_env LEGACY_BACKFILL_ANCHOR_ASSET_ID
    run_sql "${REPO_ROOT}/backend/scripts/legacy-backfill-both-eyes.sql"
    ;;
  *)
    usage
    ;;
esac

echo "OK: legacy backfill case '${CASE}' applied"
