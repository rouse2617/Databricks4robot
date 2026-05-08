#!/usr/bin/env bash
# Apply incremental Postgres migrations to an *existing* database.
#
# Docker Compose mounts backend/migrations into /docker-entrypoint-initdb.d only
# when the data volume is first created; older volumes can miss newer files
# (e.g. 008 aggregate_type on asset_events). This script replays safe deltas.
#
# Skips:
#   001_init.sql  — full bootstrap (use a fresh volume for new installs)
#   002_*.sql      — (none in repo; optional dev seed is scripts/postgres/dev_seed.sql + apply_dev_seed.sh)
#   013_*.sql     — drops legacy cf_* columns (opt-in only)
#
# Usage:
#   DATABASE_URL=postgresql://user:pass@host:5432/db bash scripts/apply_pg_deltas.sh
#   APPLY_CF_LEGACY_DROP=1 ...   # also run 013_drop_cf_legacy_columns.sql
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MIGRATIONS="$(cd "${SCRIPT_DIR}/../migrations" && pwd)"
DB_URL="${DATABASE_URL:-postgresql://postgres:postgres@127.0.0.1:5432/data4cyber}"

apply_one() {
  local f="$1"
  echo "==> $(basename "$f")"
  psql "$DB_URL" -v ON_ERROR_STOP=1 -f "$f"
}

shopt -s nullglob
deltas=()
for f in "$MIGRATIONS"/*.sql; do
  b="$(basename "$f")"
  case "$b" in
    001_*) continue ;;
    002_*) continue ;;
    013_*) continue ;;
    *) deltas+=("$f") ;;
  esac
done

IFS=$'\n' sorted="$(printf '%s\n' "${deltas[@]}" | sort)"
unset IFS

while IFS= read -r f; do
  [[ -n "$f" ]] || continue
  apply_one "$f"
done <<< "$sorted"

if [[ "${APPLY_CF_LEGACY_DROP:-}" == "1" ]]; then
  apply_one "$MIGRATIONS/013_drop_cf_legacy_columns.sql"
else
  echo "==> (skip 013_drop_cf_legacy_columns.sql — set APPLY_CF_LEGACY_DROP=1 to apply)"
fi

echo "Done. Verify asset_events: psql ... -c \"\\d asset_events\""
